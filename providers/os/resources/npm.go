// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/fsutil"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/javascript/bunlock"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/javascript/packagejson"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/javascript/packagelockjson"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/javascript/pnpmlock"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/javascript/yarnlock"
	"go.mondoo.com/mql/v13/types"
)

var defaultNpmPaths = []string{
	// Linux
	"/usr/local/lib",
	"/opt/homebrew/lib",
	"/usr/lib",
	"/home/*/.npm-global/lib",
	// Windows
	"C:\\Users\\*\\AppData\\Roaming\\npm",
	"C:\\Program Files\\nodejs\\node_modules\\npm",
	"C:\\Users\\*\\node_modules",
	// macOS
	"/Users/*/.npm-global/lib",
	// Container app paths
	"/app",
	"/home/node/app",
	"/usr/src/app",
}

func initNpmPackages(_ *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		_, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in npm.packages initialization, it must be a string")
		}
	} else {
		args["path"] = llx.StringData("")
	}

	if x, ok := args["paths"]; ok {
		xv, ok := x.Value.([]any)
		if !ok {
			return nil, nil, errors.New("wrong type for 'paths' in npm.packages initialization, it must be a list of strings")
		}
		for i := range xv {
			_, ok := xv[i].(string)
			if !ok {
				return nil, nil, errors.New("wrong type for 'paths' in npm.packages initialization, it must be a list of strings")
			}
		}
	}

	if x, ok := args["contents"]; ok {
		xv, ok := x.Value.(map[string]any)
		if !ok {
			return nil, nil, errors.New("wrong type for 'contents' in npm.packages initialization, it must be a map of string to string")
		}
		for k, v := range xv {
			if _, ok := v.(string); !ok {
				return nil, nil, fmt.Errorf("wrong type for 'contents[%q]' in npm.packages initialization, values must be strings", k)
			}
		}
	}
	return args, nil, nil
}

func (r *mqlNpmPackages) id() (string, error) {
	entries, err := r.getPaths()
	if err != nil {
		return "", err
	}

	// fold content into the id so two instances with the same filenames but
	// different content do not collide in the runtime resource cache
	contents := r.contentMap()

	if len(entries) == 0 && len(contents) == 0 {
		return "npm.packages", nil
	}
	if len(contents) == 0 && len(entries) == 1 {
		return "npm.packages/" + entries[0], nil
	}

	sort.Strings(entries)
	hash := sha256.New()
	for _, entry := range entries {
		hash.Write([]byte(entry))
		hash.Write([]byte{0})
	}
	// stable iteration order: hash by sorted filename, with both key and
	// value included so different content under the same key disambiguates
	keys := make([]string, 0, len(contents))
	for k := range contents {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		hash.Write([]byte(k))
		hash.Write([]byte{0})
		hash.Write([]byte(contents[k]))
		hash.Write([]byte{0})
	}
	return "npm.packages/" + hex.EncodeToString(hash.Sum(nil)), nil
}

func (r *mqlNpmPackages) paths() ([]any, error) {
	// when content is supplied, advertise the supplied filenames rather than
	// the filesystem-default search paths so consumers see the actual inputs
	if contentKeys := r.contentKeys(); len(contentKeys) > 0 {
		res := []any{}
		for _, k := range contentKeys {
			res = append(res, k)
		}
		return res, nil
	}

	paths, err := r.getPaths()
	if err != nil {
		return nil, err
	}
	res := []any{}
	for i := range paths {
		res = append(res, paths[i])
	}
	return res, nil
}

// contentKeys returns the sorted set of filenames provided through the
// `contents` init arg, or nil if none.
func (r *mqlNpmPackages) contentKeys() []string {
	m := r.contentMap()
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// contentMap returns the `contents` init arg as map[string]string.
// The MQL compiler coerces literal map values typed as `any` to `string` at
// the resource-arg site; the init function still asserts each value's type.
func (r *mqlNpmPackages) contentMap() map[string]string {
	if r.Contents.Error != nil || len(r.Contents.Data) == 0 {
		return nil
	}
	out := make(map[string]string, len(r.Contents.Data))
	for k, v := range r.Contents.Data {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

// gatherPackagesFromSystemDefaults returns
// - direct packages
// - transitive packages
// - evidence files
func collectNpmPackagesInPaths(runtime *plugin.Runtime, fs afero.Fs, paths []string) ([]*languages.Package, []*languages.Package, []string, error) {
	var directPackageList []*languages.Package
	var transitivePackageList []*languages.Package
	evidenceFiles := []string{}

	handler := func(nodeModulesPath string) {
		// Not found is an expected error and we handle that properly
		bom, err := collectNpmPackages(runtime, fs, nodeModulesPath)
		if err != nil {
			return
		}

		root := bom.Root()
		if root != nil {
			directPackageList = append(directPackageList, root)
		}
		transitive := bom.Transitive()
		if transitive != nil {
			transitivePackageList = append(transitivePackageList, transitive...)
		}
	}

	log.Debug().Msg("searching for npm packages in default locations")
	err := fsutil.WalkGlob(fs, paths, func(fs afero.Fs, walkPath string) error {
		afs := &afero.Afero{Fs: fs}

		// check root directory
		handler(walkPath)

		// if we have a lock file, we do not need to check for node_modules directory
		if hasLockfile(runtime, fs, walkPath) {
			return nil
		}

		// when node_modules exist we check the directory for dependencies (only applies if lockfile is missing)
		nodeModulesPath := filepath.Join(walkPath, "node_modules")
		_, err := afs.Stat(nodeModulesPath)
		if err == nil {
			log.Debug().Str("path", walkPath).Msg("found npm package")
			files, err := afs.ReadDir(nodeModulesPath)
			if err != nil {
				return nil
			}
			for _, nodePkg := range files {
				p := nodePkg.Name()

				// we ignore the files
				if !nodePkg.IsDir() {
					continue
				}

				// check that the directory starts with @, which is used for npm scopes
				// see https://docs.npmjs.com/about-scopes
				if strings.HasPrefix(nodePkg.Name(), "@") {
					scopePath := filepath.Join(nodeModulesPath, nodePkg.Name())
					d, err := afs.Open(scopePath)
					if err != nil {
						continue
					}
					scopedPkgs, err := d.Readdirnames(-1)
					if err != nil {
						continue
					}
					for _, scopedPkg := range scopedPkgs {
						isDir, err := afs.IsDir(filepath.Join(scopePath, scopedPkg))
						if !isDir || err != nil {
							continue
						}
						handler(filepath.Join(scopePath, scopedPkg))
					}
				} else {
					log.Debug().Str("path", p).Msg("checking for package-lock.json or package.json file")
					handler(filepath.Join(nodeModulesPath, p))
				}
			}
			return nil
		}

		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return directPackageList, transitivePackageList, evidenceFiles, nil
}

// hasLockfile checks for the lock files
func hasLockfile(runtime *plugin.Runtime, fs afero.Fs, path string) bool {
	// specific path was provided
	afs := &afero.Afero{Fs: fs}
	isDir, err := afs.IsDir(path)
	if err != nil {
		return false
	}

	searchPaths := []string{}
	if isDir {
		// check if there is a lockfile
		searchPaths = append(searchPaths,
			filepath.Join(path, "/package-lock.json"),
			filepath.Join(path, "/pnpm-lock.yaml"),
			filepath.Join(path, "/yarn.lock"),
			filepath.Join(path, "/bun.lock"),
		)
	} else if strings.HasSuffix(path, "package-lock.json") ||
		strings.HasSuffix(path, "pnpm-lock.yaml") ||
		strings.HasSuffix(path, "yarn.lock") ||
		strings.HasSuffix(path, "bun.lock") {
		searchPaths = append(searchPaths, path)
	}

	// filter out non-existing files using the new slice package
	filteredSearchPath := []string{}
	for i := range searchPaths {
		exists, _ := afs.Exists(searchPaths[i])
		if exists {
			filteredSearchPath = append(filteredSearchPath, searchPaths[i])
		}
	}
	return len(filteredSearchPath) > 0
}

// npmExtractorFor returns the parser matched by the filename suffix, or nil
// if the path does not look like a supported JS package or lock file.
func npmExtractorFor(path string) languages.Extractor {
	switch {
	case strings.HasSuffix(path, "package-lock.json"):
		return &packagelockjson.Extractor{}
	case strings.HasSuffix(path, "pnpm-lock.yaml"):
		return &pnpmlock.Extractor{}
	case strings.HasSuffix(path, "yarn.lock"):
		return &yarnlock.Extractor{}
	case strings.HasSuffix(path, "bun.lock"):
		return &bunlock.Extractor{}
	case strings.HasSuffix(path, "package.json"):
		return &packagejson.Extractor{}
	}
	return nil
}

// collectNpmPackagesFromContents parses the supplied {path: content} map
// directly, without touching the connection's filesystem. Lockfiles win
// for direct/transitive dependencies (resolved versions), but the root
// package metadata is taken from package.json when supplied — lockfiles
// don't always carry a top-level root entry (older formats, partial
// fixtures), and package.json is the authoritative source for project
// name/version anyway.
func collectNpmPackagesFromContents(contents map[string]string) (*languages.Package, []*languages.Package, []*languages.Package, []string, error) {
	type entry struct {
		path     string
		isLock   bool
		ext      languages.Extractor
		contents string
	}
	var entries []entry
	for path, content := range contents {
		ext := npmExtractorFor(path)
		if ext == nil {
			continue
		}
		isLock := !strings.HasSuffix(path, "package.json")
		entries = append(entries, entry{path: path, isLock: isLock, ext: ext, contents: content})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })

	var (
		manifestRoot, lockRoot             *languages.Package
		manifestDirect, lockDirect         []*languages.Package
		manifestTransitive, lockTransitive []*languages.Package
		manifestFiles, lockFiles           []string
	)

	for _, e := range entries {
		bom, err := e.ext.Parse(strings.NewReader(e.contents), e.path)
		if err != nil {
			log.Debug().Err(err).Str("file", e.path).Msg("failed to parse npm manifest content")
			continue
		}
		if e.isLock {
			lockFiles = append(lockFiles, e.path)
			if r := bom.Root(); r != nil && lockRoot == nil {
				lockRoot = r
			}
			lockDirect = append(lockDirect, bom.Direct()...)
			lockTransitive = append(lockTransitive, bom.Transitive()...)
		} else {
			manifestFiles = append(manifestFiles, e.path)
			if r := bom.Root(); r != nil && manifestRoot == nil {
				manifestRoot = r
			}
			manifestDirect = append(manifestDirect, bom.Direct()...)
			manifestTransitive = append(manifestTransitive, bom.Transitive()...)
		}
	}

	root := manifestRoot
	if root == nil {
		root = lockRoot
	}

	var direct, transitive []*languages.Package
	var files []string
	if len(lockFiles) > 0 {
		direct = lockDirect
		transitive = lockTransitive
		files = lockFiles
		// keep the package.json in the file list when its root contributed
		if manifestRoot != nil && root == manifestRoot {
			files = append(files, manifestFiles...)
			sort.Strings(files)
		}
	} else {
		direct = manifestDirect
		transitive = manifestTransitive
		files = manifestFiles
	}

	return root, direct, transitive, files, nil
}

func collectNpmPackages(runtime *plugin.Runtime, fs afero.Fs, path string) (languages.Bom, error) {
	// specific path was provided
	afs := &afero.Afero{Fs: fs}
	isDir, err := afs.IsDir(path)
	if err != nil {
		return nil, err
	}

	searchPaths := []string{}
	if isDir {
		// check if there is a lockfile or package.json file
		searchPaths = append(searchPaths,
			filepath.Join(path, "/package-lock.json"),
			filepath.Join(path, "/pnpm-lock.yaml"),
			filepath.Join(path, "/yarn.lock"),
			filepath.Join(path, "/bun.lock"),
			filepath.Join(path, "/package.json"),
		)
	} else if strings.HasSuffix(path, "package-lock.json") {
		searchPaths = append(searchPaths, path)
	} else if strings.HasSuffix(path, "pnpm-lock.yaml") {
		searchPaths = append(searchPaths, path)
	} else if strings.HasSuffix(path, "yarn.lock") {
		searchPaths = append(searchPaths, path)
	} else if strings.HasSuffix(path, "bun.lock") {
		searchPaths = append(searchPaths, path)
	} else if strings.HasSuffix(path, "package.json") {
		searchPaths = append(searchPaths, path)
	}

	// filter out non-existing files using the new slice package
	filteredSearchPath := []string{}
	for i := range searchPaths {
		exists, _ := afs.Exists(searchPaths[i])
		if exists {
			filteredSearchPath = append(filteredSearchPath, searchPaths[i])
		}
	}

	if len(filteredSearchPath) == 0 {
		return nil, fmt.Errorf("path %s is not a supported JavaScript lockfile or package.json", path)
	}

	// technically we should only have one file, this logic will always pick the first one
	for _, searchPath := range filteredSearchPath {
		// if there is a package-lock.json file, we use it
		f, err := newFile(runtime, searchPath)
		if err != nil {
			return nil, err
		}
		content := f.GetContent()
		if content.Error != nil {
			return nil, content.Error
		}

		var extractor languages.Extractor

		if strings.HasSuffix(searchPath, "package-lock.json") {
			extractor = &packagelockjson.Extractor{}
		} else if strings.HasSuffix(searchPath, "pnpm-lock.yaml") {
			extractor = &pnpmlock.Extractor{}
		} else if strings.HasSuffix(searchPath, "yarn.lock") {
			extractor = &yarnlock.Extractor{}
		} else if strings.HasSuffix(searchPath, "bun.lock") {
			extractor = &bunlock.Extractor{}
		} else if strings.HasSuffix(searchPath, "package.json") {
			extractor = &packagejson.Extractor{}
		}

		if extractor != nil {
			return extractor.Parse(strings.NewReader(content.Data), searchPath)
		}
	}

	return nil, errors.New("could not parse package-lock.json or package.json file")
}

type mqlNpmPackagesInternal struct {
	mutex sync.Mutex
}

func (r *mqlNpmPackages) getPaths() ([]string, error) {
	paths := []string{}
	if r.Paths.Error != nil {
		return nil, r.Paths.Error
	}

	for i := range r.Paths.Data {
		paths = append(paths, r.Paths.Data[i].(string))
	}

	if r.Path.Error != nil {
		return nil, r.Path.Error
	}
	if r.Path.Data != "" {
		paths = append(paths, r.Path.Data)
	}

	sort.Strings(paths)
	paths = slices.Compact(paths)

	if len(paths) == 0 {
		paths = defaultNpmPaths
	}
	return paths, nil
}

func (r *mqlNpmPackages) gatherData() error {
	// ensure we only gather data once, happens when multiple fields are called by MQL
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var root *languages.Package
	var directDependencies []*languages.Package
	var transitiveDependencies []*languages.Package
	var filePaths []string

	// content-map branch: parse provided file contents directly, without
	// touching the os connection. Used when running on a non-os connection
	// (for example, feeding `github.file.content` from a github repo scan).
	if contents := r.contentMap(); len(contents) > 0 {
		var err error
		root, directDependencies, transitiveDependencies, filePaths, err = collectNpmPackagesFromContents(contents)
		if err != nil {
			return err
		}
		return r.finalizeData(root, directDependencies, transitiveDependencies, filePaths)
	}

	// NOTE: that we do not get paths that are an empty slice here
	paths, err := r.getPaths()
	if err != nil {
		return err
	}

	// we check if the path is a directory or a file
	// if it is a directory, we check if there is a package-lock.json or package.json file
	conn := r.MqlRuntime.Connection.(shared.Connection)

	fs := conn.FileSystem()

	if len(paths) > 1 {
		// no specific path was provided, we search through default locations
		// here we are not going to have a root package, only direct and transitive dependencies
		directDependencies, transitiveDependencies, filePaths, err = collectNpmPackagesInPaths(r.MqlRuntime, fs, paths)
		if err != nil {
			return err
		}
	} else {
		// do not load anything if the path does not exist
		_, err := fs.Stat(paths[0])
		if err == nil {
			// specific path was provided and most likely it is a package-lock.json or package.json file or a directory
			// that contains one of those files. We will have a root package direct and transitive dependencies
			bom, err := collectNpmPackages(r.MqlRuntime, fs, paths[0])
			if err != nil {
				return err
			}
			filePaths = append(filePaths, paths[0])
			root = bom.Root()
			directDependencies = bom.Direct()
			transitiveDependencies = bom.Transitive()
		}
	}

	return r.finalizeData(root, directDependencies, transitiveDependencies, filePaths)
}

func (r *mqlNpmPackages) finalizeData(root *languages.Package, directDependencies, transitiveDependencies []*languages.Package, filePaths []string) error {
	// sort packages by name
	slices.SortFunc(directDependencies, languages.SortFn)
	slices.SortFunc(transitiveDependencies, languages.SortFn)

	if root != nil {
		mqlPkg, err := newNpmPackage(r.MqlRuntime, root)
		if err != nil {
			return err
		}
		r.Root = plugin.TValue[*mqlNpmPackage]{Data: mqlPkg, State: plugin.StateIsSet}
	} else {
		r.Root = plugin.TValue[*mqlNpmPackage]{State: plugin.StateIsSet | plugin.StateIsNull}
	}

	// create a resource for each package
	transitiveResources, err := newNpmPackageList(r.MqlRuntime, transitiveDependencies)
	if err != nil {
		return err
	}
	r.List = plugin.TValue[[]any]{Data: transitiveResources, State: plugin.StateIsSet}

	directResources, err := newNpmPackageList(r.MqlRuntime, directDependencies)
	if err != nil {
		return err
	}
	r.DirectDependencies = plugin.TValue[[]any]{Data: directResources, State: plugin.StateIsSet}

	// create files for each path
	mqlFiles := []any{}
	for i := range filePaths {
		path := filePaths[i]
		lf, err := CreateResource(r.MqlRuntime, "pkgFileInfo", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return err
		}
		mqlFiles = append(mqlFiles, lf)
	}
	r.Files = plugin.TValue[[]any]{Data: mqlFiles, State: plugin.StateIsSet}

	return nil
}

func (r *mqlNpmPackages) root() (*mqlNpmPackage, error) {
	return nil, r.gatherData()
}

func (r *mqlNpmPackages) directDependencies() ([]any, error) {
	return nil, r.gatherData()
}

func (r *mqlNpmPackages) list() ([]any, error) {
	return nil, r.gatherData()
}

func (r *mqlNpmPackages) files() ([]any, error) {
	return nil, r.gatherData()
}

func (r *mqlNpmPackages) scripts() (map[string]any, error) {
	type packageJson struct {
		Scripts map[string]string `json:"scripts"`
	}

	// when content is supplied, parse scripts out of the package.json entry
	// (if any) without touching the filesystem
	if contents := r.contentMap(); len(contents) > 0 {
		var raw string
		for path, c := range contents {
			if strings.HasSuffix(path, "package.json") {
				raw = c
				break
			}
		}
		if raw == "" {
			return map[string]any{}, nil
		}
		pkgInfo := packageJson{}
		if err := json.Unmarshal([]byte(raw), &pkgInfo); err != nil {
			return nil, err
		}
		res := make(map[string]any)
		for k, v := range pkgInfo.Scripts {
			res[k] = v
		}
		return res, nil
	}

	if r.Path.Error != nil {
		return nil, r.Path.Error
	}
	path := r.Path.Data

	f, err := newFile(r.MqlRuntime, path)
	if err != nil {
		return nil, err
	}
	content := f.GetContent()
	if content.Error != nil {
		return nil, content.Error
	}

	pkgInfo := packageJson{}
	err = json.Unmarshal([]byte(content.Data), &pkgInfo)
	if err != nil {
		return nil, err
	}

	res := make(map[string]any)
	for k, v := range pkgInfo.Scripts {
		res[k] = v
	}
	return res, nil
}

// newNpmPackageList creates a list of npm package resources
func newNpmPackageList(runtime *plugin.Runtime, packages []*languages.Package) ([]any, error) {
	resources := []any{}
	for i := range packages {
		pkg, err := newNpmPackage(runtime, packages[i])
		if err != nil {
			return nil, err
		}
		resources = append(resources, pkg)
	}
	return resources, nil
}

// newNpmPackage creates a new npm package resource
func newNpmPackage(runtime *plugin.Runtime, pkg *languages.Package) (*mqlNpmPackage, error) {
	// handle cpes
	cpes := []any{}
	for i := range pkg.Cpes {
		cpe, err := runtime.CreateSharedResource("cpe", map[string]*llx.RawData{
			"uri": llx.StringData(pkg.Cpes[i]),
		})
		if err != nil {
			return nil, err
		}
		cpes = append(cpes, cpe)
	}

	// create files for each path
	mqlFiles := []any{}
	for i := range pkg.EvidenceList {
		evidence := pkg.EvidenceList[i]
		lf, err := CreateResource(runtime, "pkgFileInfo", map[string]*llx.RawData{
			"path": llx.StringData(evidence.Value),
		})
		if err != nil {
			return nil, err
		}
		mqlFiles = append(mqlFiles, lf)
	}

	path := ""
	if len(mqlFiles) > 0 {
		if fi, ok := mqlFiles[0].(*mqlPkgFileInfo); ok {
			path = fi.Path.Data
		}
	}
	mqlPkg, err := CreateResource(runtime, "npm.package", map[string]*llx.RawData{
		"id":          llx.StringData(pkg.Name + path),
		"name":        llx.StringData(pkg.Name),
		"version":     llx.StringData(pkg.Version),
		"purl":        llx.StringData(pkg.Purl),
		"cpes":        llx.ArrayData(cpes, types.Resource("cpe")),
		"files":       llx.ArrayData(mqlFiles, types.Resource("pkgFileInfo")),
		"description": llx.StringData(pkg.Description),
		"author":      llx.StringData(pkg.Author),
		"license":     llx.StringData(pkg.License),
	})
	if err != nil {
		return nil, err
	}
	return mqlPkg.(*mqlNpmPackage), nil
}

func (k *mqlNpmPackage) id() (string, error) {
	return k.Id.Data, nil
}

func (r *mqlNpmPackage) name() (string, error) {
	return "", r.populateData()
}

func (r *mqlNpmPackage) version() (string, error) {
	return "", r.populateData()
}

func (r *mqlNpmPackage) purl() (string, error) {
	return "", r.populateData()
}

func (r *mqlNpmPackage) cpes() ([]any, error) {
	return nil, r.populateData()
}

func (r *mqlNpmPackage) files() ([]any, error) {
	return nil, errors.New("not implemented")
}

func (r *mqlNpmPackage) populateData() error {
	// future iterations will read a npm package.json file and populate the data
	// all data is already available in the package object
	r.Name = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Version = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Purl = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Cpes = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Files = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
	return nil
}
