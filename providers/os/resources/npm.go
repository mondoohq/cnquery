// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/fsutil"
	"go.mondoo.com/cnquery/v12/providers/os/resources/languages"
	"go.mondoo.com/cnquery/v12/providers/os/resources/languages/javascript/packagejson"
	"go.mondoo.com/cnquery/v12/providers/os/resources/languages/javascript/packagelockjson"
	"go.mondoo.com/cnquery/v12/types"
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
			return nil, nil, errors.New("Wrong type for 'path' in npm initialization, it must be a string")
		}
	} else {
		// empty path means search through default locations
		args["path"] = llx.StringData("")
	}

	return args, nil, nil
}

func (r *mqlNpmPackages) id() (string, error) {
	path := r.Path.Data
	if path == "" {
		return "npm.packages", nil
	}

	return "npm.packages/" + path, nil
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

		// we walk through the directories and check if there is a node_modules directory
		log.Debug().Str("path", walkPath).Msg("found npm package")
		nodeModulesPath := filepath.Join(walkPath, "node_modules")

		files, err := afs.ReadDir(nodeModulesPath)
		if err != nil {
			// we ignore the error, it is expected that there is no node_modules directory
			return nil
		}
		for i := range files {
			f := files[i]
			p := f.Name()

			if !f.IsDir() {
				continue
			}

			log.Debug().Str("path", p).Msg("checking for package-lock.json or package.json file")
			handler(filepath.Join(nodeModulesPath, p))
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
		// check if there is a package-lock.json or package.json file
		searchPaths = append(searchPaths, filepath.Join(path, "/package-lock.json"))
	} else if strings.HasSuffix(path, "package-lock.json") {
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

func collectNpmPackages(runtime *plugin.Runtime, fs afero.Fs, path string) (languages.Bom, error) {
	// specific path was provided
	afs := &afero.Afero{Fs: fs}
	isDir, err := afs.IsDir(path)
	if err != nil {
		return nil, err
	}

	searchPaths := []string{}
	if isDir {
		// check if there is a package-lock.json or package.json file
		searchPaths = append(searchPaths, filepath.Join(path, "/package-lock.json"), filepath.Join(path, "/package.json"))
	} else if strings.HasSuffix(path, "package-lock.json") {
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
		return nil, fmt.Errorf("path %s is not a package.json or package-lock.json file", path)
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

func (r *mqlNpmPackages) gatherData() error {
	// ensure we only gather data once, happens when multiple fields are called by MQL
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.Path.Error != nil {
		return r.Path.Error
	}
	path := r.Path.Data

	// we check if the path is a directory or a file
	// if it is a directory, we check if there is a package-lock.json or package.json file
	conn := r.MqlRuntime.Connection.(shared.Connection)

	var root *languages.Package
	var directDependencies []*languages.Package
	var transitiveDependencies []*languages.Package
	var filePaths []string
	var err error

	if path == "" {
		// no specific path was provided, we search through default locations
		// here we are not going to have a root package, only direct and transitive dependencies
		directDependencies, transitiveDependencies, filePaths, err = collectNpmPackagesInPaths(r.MqlRuntime, conn.FileSystem(), defaultNpmPaths)
		if err != nil {
			return err
		}
	} else {
		// specific path was provided and most likely it is a package-lock.json or package.json file or a directory
		// that contains one of those files. We will have a root package direct and transitive dependencies
		bom, err := collectNpmPackages(r.MqlRuntime, conn.FileSystem(), path)
		if err != nil {
			return err
		}
		root = bom.Root()
		directDependencies = bom.Direct()
		transitiveDependencies = bom.Transitive()
	}

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

	type packageJson struct {
		Scripts map[string]string `json:"scripts"`
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
		"id":      llx.StringData(pkg.Name + path),
		"name":    llx.StringData(pkg.Name),
		"version": llx.StringData(pkg.Version),
		"purl":    llx.StringData(pkg.Purl),
		"cpes":    llx.ArrayData(cpes, types.Resource("cpe")),
		"files":   llx.ArrayData(mqlFiles, types.Resource("pkgFileInfo")),
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
