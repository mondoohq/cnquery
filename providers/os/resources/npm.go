// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"cmp"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/fsutil"
	"go.mondoo.com/cnquery/v11/providers/os/resources/languages"
	"go.mondoo.com/cnquery/v11/providers/os/resources/languages/javascript/packagejson"
	"go.mondoo.com/cnquery/v11/providers/os/resources/languages/javascript/packagelockjson"
	"go.mondoo.com/cnquery/v11/sbom"
	"go.mondoo.com/cnquery/v11/types"
)

var (
	defaultNpmPaths = []string{
		// Linux
		"/usr/local/lib",
		"/opt/homebrew/lib",
		"/usr/lib",
		"/home/*/.npm-global/lib",
		// Windows
		"C:\\Users\\*\\AppData\\Roaming\\npm",
		// macOS
		"/Users/*/.npm-global/lib",
	}
)

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
func collectNpmPackagesInPaths(runtime *plugin.Runtime, fs afero.Fs, paths []string) ([]*sbom.Package, []*sbom.Package, []string, error) {
	var directPackageList []*sbom.Package
	var transitivePackageList []*sbom.Package
	evidenceFiles := []string{}

	log.Debug().Msg("searching for npm packages in default locations")
	err := fsutil.WalkGlob(fs, paths, func(fs afero.Fs, walkPath string) error {
		afs := &afero.Afero{Fs: fs}

		// we walk through the directories and check if there is a node_modules directory
		log.Debug().Str("path", walkPath).Msg("found npm package")
		nodeModulesPath := filepath.Join(walkPath, "node_modules")
		var files, err = afs.ReadDir(nodeModulesPath)
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

			// Not found is an expected error and we handle that properly
			bom, err := collectNpmPackages(runtime, fs, filepath.Join(nodeModulesPath, p))
			if err != nil {
				continue
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
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return directPackageList, transitivePackageList, evidenceFiles, nil
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
		} else {
			return nil, errors.New("could not find suitable extractor for file: " + searchPath)
		}

		return extractor.Parse(strings.NewReader(content.Data), searchPath)
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

	var root *sbom.Package
	var directDependencies []*sbom.Package
	var transitiveDependencies []*sbom.Package
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
	sortFn := func(a, b *sbom.Package) int {
		if n := cmp.Compare(a.Name, b.Name); n != 0 {
			return n
		}
		// if names are equal, order by version
		return cmp.Compare(a.Version, b.Version)
	}
	slices.SortFunc(directDependencies, sortFn)
	slices.SortFunc(transitiveDependencies, sortFn)

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
	transitiveResources := []interface{}{}
	for i := range transitiveDependencies {
		newNpmPackages, err := newNpmPackage(r.MqlRuntime, transitiveDependencies[i])
		if err != nil {
			return err
		}
		transitiveResources = append(transitiveResources, newNpmPackages)
	}
	r.List = plugin.TValue[[]interface{}]{Data: transitiveResources, State: plugin.StateIsSet}

	directResources := []interface{}{}
	for i := range directDependencies {
		newNpmPackages, err := newNpmPackage(r.MqlRuntime, directDependencies[i])
		if err != nil {
			return err
		}
		directResources = append(directResources, newNpmPackages)
	}
	r.DirectDependencies = plugin.TValue[[]interface{}]{Data: directResources, State: plugin.StateIsSet}

	// create files for each path
	mqlFiles := []interface{}{}
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
	r.Files = plugin.TValue[[]interface{}]{Data: mqlFiles, State: plugin.StateIsSet}

	return nil
}

func (r *mqlNpmPackages) root() (*mqlNpmPackage, error) {
	return nil, r.gatherData()
}

func (r *mqlNpmPackages) directDependencies() ([]interface{}, error) {
	return nil, r.gatherData()
}

func (r *mqlNpmPackages) list() ([]interface{}, error) {
	return nil, r.gatherData()
}

func (r *mqlNpmPackages) files() ([]interface{}, error) {
	return nil, r.gatherData()
}

// newNpmPackage creates a new npm package resource
func newNpmPackage(runtime *plugin.Runtime, pkg *sbom.Package) (*mqlNpmPackage, error) {
	// handle cpes
	cpes := []interface{}{}
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
	mqlFiles := []interface{}{}
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

	mqlPkg, err := CreateResource(runtime, "npm.package", map[string]*llx.RawData{
		"id":      llx.StringData(pkg.Name),
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

func (r *mqlNpmPackage) cpes() ([]interface{}, error) {
	return nil, r.populateData()
}

func (r *mqlNpmPackage) files() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (r *mqlNpmPackage) populateData() error {
	// future iterations will read a npm package.json file and populate the data
	// all data is already available in the package object
	r.Name = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Version = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Purl = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Cpes = plugin.TValue[[]interface{}]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Files = plugin.TValue[[]interface{}]{State: plugin.StateIsSet | plugin.StateIsNull}
	return nil
}
