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
	"go.mondoo.com/cnquery/v11/providers/os/resources/languages"
	"go.mondoo.com/cnquery/v11/providers/os/resources/languages/javascript/packagejson"
	"go.mondoo.com/cnquery/v11/providers/os/resources/languages/javascript/packagelockjson"
	"go.mondoo.com/cnquery/v11/sbom"
	"go.mondoo.com/cnquery/v11/types"
)

func initNpmPackages(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func getFileContent(runtime *plugin.Runtime, path string) (*mqlFile, error) {
	f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}
	file := f.(*mqlFile)
	return file, nil
}

var (
	// we need to add extra testing for windows paths
	//windowsDefaultNpmPaths = []string{
	//	"C:\\Users\\%\\AppData\\Roaming\\npm",
	//}
	linuxDefaultNpmPaths = []string{
		"/usr/local/lib",
		"/opt/homebrew/lib",
		"/usr/lib",
		"/home/%/.npm-global/lib",
		"/Users/%/.npm-global/lib",
	}
)

func (r *mqlNpmPackages) gatherPackagesFromSystemDefaults(conn shared.Connection) ([]*sbom.Package, []*sbom.Package, []string, error) {
	var directPackageList []*sbom.Package
	var transitivePackageList []*sbom.Package
	evidenceFiles := []string{}
	log.Debug().Msg("searching for npm packages in default locations")
	afs := &afero.Afero{Fs: conn.FileSystem()}
	// we search through default system locations
	for _, pattern := range linuxDefaultNpmPaths {
		log.Debug().Str("path", pattern).Msg("searching for npm packages")
		m, err := afero.Glob(conn.FileSystem(), pattern)
		if err != nil {
			log.Debug().Err(err).Str("path", pattern).Msg("could not search for npm packages")
			// nothing to do, we just ignore it
		}
		for _, walkPath := range m {
			// we walk through the directories and check if there is a node_modules directory
			log.Debug().Str("path", walkPath).Msg("found npm package")
			nodeModulesPath := filepath.Join(walkPath, "node_modules")
			var files, err = afs.ReadDir(nodeModulesPath)
			if err != nil {
				continue
			}
			for i := range files {
				f := files[i]
				p := f.Name()

				if !f.IsDir() {
					continue
				}

				log.Debug().Str("path", p).Msg("checking for package-lock.json or package.json file")

				// check if there is a package-lock.json or package.json file
				packageLockPath := filepath.Join(nodeModulesPath, p, "/package-lock.json")
				packageJsonPath := filepath.Join(nodeModulesPath, p, "/package.json")

				packageLockExists, _ := afs.Exists(packageLockPath)
				packageJsonExists, _ := afs.Exists(packageJsonPath)

				// add files to evidence
				if packageLockExists {
					evidenceFiles = append(evidenceFiles, packageLockPath)
				}
				if packageJsonExists {
					evidenceFiles = append(evidenceFiles, packageJsonPath)
				}

				// parse npm files
				if packageLockExists {
					log.Debug().Str("path", packageLockPath).Msg("found package-lock.json file")
					f, err := getFileContent(r.MqlRuntime, packageLockPath)
					if err != nil {
						continue
					}
					content := f.GetContent()
					if content.Error != nil {
						continue
					}

					p := &packagelockjson.Extractor{}
					info, err := p.Parse(strings.NewReader(content.Data), packageLockPath)
					if err != nil {
						log.Error().Err(err).Str("path", packageLockPath).Msg("could not parse package-lock.json file")
					}
					root := info.Root()
					if root != nil {
						directPackageList = append(directPackageList, root)
					}
					transitive := info.Transitive()
					if transitive != nil {
						transitivePackageList = append(transitivePackageList, transitive...)
					}

				} else if packageJsonExists {
					log.Debug().Str("path", packageJsonPath).Msg("found package.json file")
					f, err := getFileContent(r.MqlRuntime, packageJsonPath)
					if err != nil {
						continue
					}
					content := f.GetContent()
					if content.Error != nil {
						continue
					}

					p := &packagejson.Extractor{}
					info, err := p.Parse(strings.NewReader(content.Data), packageJsonPath)
					if err != nil {
						log.Error().Err(err).Str("path", packageJsonPath).Msg("could not parse package.json file")
					}
					root := info.Root()
					if root != nil {
						directPackageList = append(directPackageList, root)
					}
					transitive := info.Transitive()
					if transitive != nil {
						transitivePackageList = append(transitivePackageList, transitive...)
					}
				}
			}
		}
	}
	return directPackageList, transitivePackageList, evidenceFiles, nil
}

func (r *mqlNpmPackages) gatherPackagesFromLocation(conn shared.Connection, path string) (*sbom.Package, []*sbom.Package, []*sbom.Package, []string, error) {
	evidenceFiles := []string{}

	// specific path was provided
	afs := &afero.Afero{Fs: conn.FileSystem()}
	isDir, err := afs.IsDir(path)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	loadPackageLock := false
	packageLockPath := ""
	loadPackageJson := false
	packageJsonPath := ""

	if isDir {
		// check if there is a package-lock.json or package.json file
		packageLockPath = filepath.Join(path, "/package-lock.json")
		packageJsonPath = filepath.Join(path, "/package.json")
	} else {
		loadPackageJson = strings.HasSuffix(path, "package-lock.json")
		if loadPackageJson {
			packageLockPath = path
		}
		loadPackageLock = strings.HasSuffix(path, "package.json")
		if loadPackageLock {
			packageJsonPath = path
		}

		if !loadPackageJson && !loadPackageLock {
			return nil, nil, nil, nil, fmt.Errorf("path %s is not a package.json or package-lock.json file", path)
		}
	}

	loadPackageLock, _ = afs.Exists(packageLockPath)
	loadPackageJson, _ = afs.Exists(packageJsonPath)

	if !loadPackageLock && !loadPackageJson {
		return nil, nil, nil, nil, fmt.Errorf("path %s does not contain a package-lock.json or package.json file", path)
	}

	// add source files as evidence to files list
	if loadPackageLock {
		evidenceFiles = append(evidenceFiles, packageLockPath)
	}
	if loadPackageJson {
		evidenceFiles = append(evidenceFiles, packageJsonPath)
	}

	// parse npm files
	var info languages.Bom
	if loadPackageLock {
		// if there is a package-lock.json file, we use it
		f, err := getFileContent(r.MqlRuntime, packageLockPath)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		content := f.GetContent()
		if content.Error != nil {
			return nil, nil, nil, nil, content.Error
		}

		p := &packagelockjson.Extractor{}
		info, err = p.Parse(strings.NewReader(content.Data), packageLockPath)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	} else if loadPackageJson {
		// if there is a package.json file, we use it
		f, err := getFileContent(r.MqlRuntime, packageJsonPath)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		content := f.GetContent()
		if content.Error != nil {
			return nil, nil, nil, nil, content.Error
		}

		p := &packagejson.Extractor{}
		info, err = p.Parse(strings.NewReader(content.Data), packageJsonPath)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	} else {
		return nil, nil, nil, nil, errors.New("could not parse package-lock.json or package.json file")
	}

	return info.Root(), info.Direct(), info.Transitive(), evidenceFiles, nil
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
		directDependencies, transitiveDependencies, filePaths, err = r.gatherPackagesFromSystemDefaults(conn)
	} else {
		// specific path was provided and most likely it is a package-lock.json or package.json file or a directory
		// that contains one of those files. We will have a root package direct and transitive dependencies
		root, directDependencies, transitiveDependencies, filePaths, err = r.gatherPackagesFromLocation(conn, path)
	}

	if err != nil {
		return err
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
		mqlPkg, err := newNpmPackages(r.MqlRuntime, root)
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
		newNpmPackages, err := newNpmPackages(r.MqlRuntime, transitiveDependencies[i])
		if err != nil {
			return err
		}
		transitiveResources = append(transitiveResources, newNpmPackages)
	}
	r.List = plugin.TValue[[]interface{}]{Data: transitiveResources, State: plugin.StateIsSet}

	directResources := []interface{}{}
	for i := range directDependencies {
		newNpmPackages, err := newNpmPackages(r.MqlRuntime, directDependencies[i])
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

func newNpmPackages(runtime *plugin.Runtime, pkg *sbom.Package) (*mqlNpmPackage, error) {
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
	// future iterations will read an npm package.json file and populate the data
	// all data is already available in the package object
	r.Name = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Version = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Purl = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Cpes = plugin.TValue[[]interface{}]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.Files = plugin.TValue[[]interface{}]{State: plugin.StateIsSet | plugin.StateIsNull}
	return nil
}
