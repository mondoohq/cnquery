// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/java/gradlelockfile"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/java/jarscanner"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/java/pomxml"
	"go.mondoo.com/mql/v13/types"
)

// defaultJavaPaths are searched for pom.xml, gradle.lockfile, and JAR files.
// Only top-level files in these directories are scanned; use java.packages(path: "/specific/dir")
// for recursive or targeted scanning.
var defaultJavaPaths = []string{
	// Common deployment locations
	"/app",
	"/opt",
	"/usr/local/lib",
	"/usr/share/java",
	// Container app paths
	"/usr/src/app",
	"/home/*/app",
}

func initJavaPackages(_ *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		_, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in java.packages initialization, it must be a string")
		}
	} else {
		args["path"] = llx.StringData("")
	}
	return args, nil, nil
}

func (r *mqlJavaPackages) id() (string, error) {
	if r.Path.Data != "" {
		return "java.packages/" + r.Path.Data, nil
	}
	return "java.packages", nil
}

type mqlJavaPackagesInternal struct {
	mutex   sync.Mutex
	fetched bool
}

func (r *mqlJavaPackages) gatherData() error {
	if r.fetched {
		return nil
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.fetched {
		return nil
	}

	conn := r.MqlRuntime.Connection.(shared.Connection)
	fs := conn.FileSystem()
	afs := &afero.Afero{Fs: fs}

	path := r.Path.Data

	var root *languages.Package
	var directDeps []*languages.Package
	var transitiveDeps []*languages.Package
	var filePaths []string

	if path != "" {
		r, d, t, f := collectJavaPackages(afs, path)
		root = r
		directDeps = d
		transitiveDeps = t
		filePaths = f
	} else {
		// Search default locations
		for _, searchPath := range defaultJavaPaths {
			// Look for pom.xml
			matches, err := afero.Glob(fs, filepath.Join(searchPath, "pom.xml"))
			if err == nil {
				for _, match := range matches {
					_, d, t, f := collectJavaPackages(afs, match)
					directDeps = append(directDeps, d...)
					transitiveDeps = append(transitiveDeps, t...)
					filePaths = append(filePaths, f...)
				}
			}

			// Look for gradle.lockfile
			matches, err = afero.Glob(fs, filepath.Join(searchPath, "gradle.lockfile"))
			if err == nil {
				for _, match := range matches {
					_, _, t, f := collectJavaPackages(afs, match)
					transitiveDeps = append(transitiveDeps, t...)
					filePaths = append(filePaths, f...)
				}
			}

			// Look for JAR files
			matches, err = afero.Glob(fs, filepath.Join(searchPath, "*.jar"))
			if err == nil {
				for _, match := range matches {
					_, _, t, f := collectJavaPackages(afs, match)
					transitiveDeps = append(transitiveDeps, t...)
					filePaths = append(filePaths, f...)
				}
			}
		}
	}

	// Sort packages
	slices.SortFunc(directDeps, languages.SortFn)
	slices.SortFunc(transitiveDeps, languages.SortFn)

	// Set root
	if root != nil {
		mqlPkg, err := newJavaPackage(r.MqlRuntime, root)
		if err != nil {
			return err
		}
		r.Root = plugin.TValue[*mqlJavaPackage]{Data: mqlPkg, State: plugin.StateIsSet}
	} else {
		r.Root = plugin.TValue[*mqlJavaPackage]{State: plugin.StateIsSet | plugin.StateIsNull}
	}

	// Set list as the union of all packages (transitive already includes direct
	// for most formats; for JAR scanning, packages go into transitiveDeps only).
	// Deduplicate by name@version.
	allPkgs := deduplicatePackages(append(transitiveDeps, directDeps...))
	slices.SortFunc(allPkgs, languages.SortFn)
	allResources, err := newJavaPackageList(r.MqlRuntime, allPkgs)
	if err != nil {
		return err
	}
	r.List = plugin.TValue[[]any]{Data: allResources, State: plugin.StateIsSet}

	// Set direct dependencies
	directResources, err := newJavaPackageList(r.MqlRuntime, directDeps)
	if err != nil {
		return err
	}
	r.DirectDependencies = plugin.TValue[[]any]{Data: directResources, State: plugin.StateIsSet}

	// Set files
	mqlFiles := []any{}
	for _, p := range filePaths {
		lf, err := CreateResource(r.MqlRuntime, "pkgFileInfo", map[string]*llx.RawData{
			"path": llx.StringData(p),
		})
		if err != nil {
			return err
		}
		mqlFiles = append(mqlFiles, lf)
	}
	r.Files = plugin.TValue[[]any]{Data: mqlFiles, State: plugin.StateIsSet}

	r.fetched = true
	return nil
}

// collectJavaPackages parses Java package metadata from a given path.
// Returns root, direct deps, transitive deps, and evidence file paths.
func collectJavaPackages(afs *afero.Afero, path string) (*languages.Package, []*languages.Package, []*languages.Package, []string) {
	isDir, err := afs.IsDir(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("could not check Java path")
		return nil, nil, nil, nil
	}

	if isDir {
		return collectJavaFromDir(afs, path)
	}

	// Single file
	if strings.HasSuffix(path, "pom.xml") {
		return collectFromPomXml(afs, path)
	}
	if strings.HasSuffix(path, "gradle.lockfile") {
		return collectFromGradleLockfile(afs, path)
	}
	if jarscanner.IsArchive(path) {
		return collectFromArchive(afs, path)
	}

	return nil, nil, nil, nil
}

func collectJavaFromDir(afs *afero.Afero, dir string) (*languages.Package, []*languages.Package, []*languages.Package, []string) {
	var root *languages.Package
	var direct []*languages.Package
	var transitive []*languages.Package
	var files []string

	// Try pom.xml first
	pomPath := filepath.Join(dir, "pom.xml")
	if exists, _ := afs.Exists(pomPath); exists {
		r, d, t, f := collectFromPomXml(afs, pomPath)
		root = r
		direct = append(direct, d...)
		transitive = append(transitive, t...)
		files = append(files, f...)
	}

	// Try gradle.lockfile
	gradlePath := filepath.Join(dir, "gradle.lockfile")
	if exists, _ := afs.Exists(gradlePath); exists {
		_, _, t, f := collectFromGradleLockfile(afs, gradlePath)
		transitive = append(transitive, t...)
		files = append(files, f...)
	}

	// Scan JAR files in the directory
	entries, err := afs.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !jarscanner.IsArchive(entry.Name()) {
				continue
			}
			jarPath := filepath.Join(dir, entry.Name())
			_, _, t, f := collectFromArchive(afs, jarPath)
			transitive = append(transitive, t...)
			files = append(files, f...)
		}
	}

	return root, direct, transitive, files
}

func collectFromPomXml(afs *afero.Afero, path string) (*languages.Package, []*languages.Package, []*languages.Package, []string) {
	f, err := afs.Open(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("could not open pom.xml")
		return nil, nil, nil, nil
	}
	defer f.Close()

	extractor := &pomxml.Extractor{}
	bom, err := extractor.Parse(f, path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("could not parse pom.xml")
		return nil, nil, nil, nil
	}

	return bom.Root(), bom.Direct(), bom.Transitive(), []string{path}
}

func collectFromGradleLockfile(afs *afero.Afero, path string) (*languages.Package, []*languages.Package, []*languages.Package, []string) {
	f, err := afs.Open(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("could not open gradle.lockfile")
		return nil, nil, nil, nil
	}
	defer f.Close()

	extractor := &gradlelockfile.Extractor{}
	bom, err := extractor.Parse(f, path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("could not parse gradle.lockfile")
		return nil, nil, nil, nil
	}

	return bom.Root(), bom.Direct(), bom.Transitive(), []string{path}
}

func collectFromArchive(afs *afero.Afero, path string) (*languages.Package, []*languages.Package, []*languages.Package, []string) {
	packages, err := jarscanner.ScanArchive(afs, path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("could not scan Java archive")
		return nil, nil, nil, nil
	}
	if len(packages) == 0 {
		return nil, nil, nil, nil
	}

	return nil, nil, packages, []string{path}
}

// deduplicatePackages removes duplicate packages by name@version.
func deduplicatePackages(pkgs []*languages.Package) []*languages.Package {
	seen := make(map[string]bool)
	var result []*languages.Package
	for _, pkg := range pkgs {
		key := pkg.Name + "@" + pkg.Version
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, pkg)
	}
	return result
}

func (r *mqlJavaPackages) root() (*mqlJavaPackage, error) {
	return nil, r.gatherData()
}

func (r *mqlJavaPackages) directDependencies() ([]any, error) {
	return nil, r.gatherData()
}

func (r *mqlJavaPackages) list() ([]any, error) {
	return nil, r.gatherData()
}

func (r *mqlJavaPackages) files() ([]any, error) {
	return nil, r.gatherData()
}

// newJavaPackageList creates a list of Java package resources.
func newJavaPackageList(runtime *plugin.Runtime, packages []*languages.Package) ([]any, error) {
	resources := []any{}
	for i := range packages {
		pkg, err := newJavaPackage(runtime, packages[i])
		if err != nil {
			return nil, err
		}
		resources = append(resources, pkg)
	}
	return resources, nil
}

// newJavaPackage creates a new Java package resource.
func newJavaPackage(runtime *plugin.Runtime, pkg *languages.Package) (*mqlJavaPackage, error) {
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

	mqlPkg, err := CreateResource(runtime, "java.package", map[string]*llx.RawData{
		"id":      llx.StringData(pkg.Name + "@" + pkg.Version + ":" + path),
		"name":    llx.StringData(pkg.Name),
		"version": llx.StringData(pkg.Version),
		"purl":    llx.StringData(pkg.Purl),
		"cpes":    llx.ArrayData(cpes, types.Resource("cpe")),
		"files":   llx.ArrayData(mqlFiles, types.Resource("pkgFileInfo")),
	})
	if err != nil {
		return nil, err
	}
	return mqlPkg.(*mqlJavaPackage), nil
}

func (k *mqlJavaPackage) id() (string, error) {
	return k.Id.Data, nil
}

func (r *mqlJavaPackage) name() (string, error) {
	return "", r.populateData()
}

func (r *mqlJavaPackage) version() (string, error) {
	return "", r.populateData()
}

func (r *mqlJavaPackage) purl() (string, error) {
	return "", r.populateData()
}

func (r *mqlJavaPackage) cpes() ([]any, error) {
	return nil, r.populateData()
}

func (r *mqlJavaPackage) files() ([]any, error) {
	return nil, r.populateData()
}

func (r *mqlJavaPackage) populateData() error {
	// java.package instances are only created via newJavaPackage, which pre-populates
	// all fields at creation time. This fallback is only reached if a java.package is
	// resolved by ID alone without going through newJavaPackage.
	r.Name = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Version = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Purl = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Cpes = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Files = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
	return nil
}
