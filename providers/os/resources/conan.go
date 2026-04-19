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
	"go.mondoo.com/mql/v13/providers/os/resources/languages/conan/conanlock"
	"go.mondoo.com/mql/v13/types"
)

// defaultConanPaths are common container app directories where conan.lock may be found.
// For project-level scanning, use conan.packages(path: "/path/to/project") explicitly.
var defaultConanPaths = []string{
	"/app",
	"/usr/src/app",
	"/home/*/app",
	"/src",
	"/workspace",
}

func initConanPackages(_ *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		_, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in conan.packages initialization, it must be a string")
		}
	} else {
		args["path"] = llx.StringData("")
	}
	return args, nil, nil
}

func (r *mqlConanPackages) id() (string, error) {
	if r.Path.Data != "" {
		return "conan.packages/" + r.Path.Data, nil
	}
	return "conan.packages", nil
}

type mqlConanPackagesInternal struct {
	mutex   sync.Mutex
	fetched bool
}

func (r *mqlConanPackages) gatherData() error {
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
		root, directDeps, transitiveDeps, filePaths = collectConanPackages(afs, fs, path)
	} else {
		for _, searchPath := range defaultConanPaths {
			matches, err := afero.Glob(fs, searchPath)
			if err != nil {
				continue
			}
			if len(matches) == 0 {
				if strings.ContainsAny(searchPath, "*?[") {
					continue
				}
				matches = []string{searchPath}
			}
			for _, match := range matches {
				collectedRoot, d, t, f := collectConanPackages(afs, fs, match)
				if collectedRoot != nil && root == nil {
					root = collectedRoot
				}
				directDeps = append(directDeps, d...)
				transitiveDeps = append(transitiveDeps, t...)
				filePaths = append(filePaths, f...)
			}
		}
	}

	// Deduplicate in case multiple paths found the same lock file
	directDeps = deduplicateConanPackages(directDeps)
	transitiveDeps = deduplicateConanPackages(transitiveDeps)
	slices.SortFunc(directDeps, languages.SortFn)
	slices.SortFunc(transitiveDeps, languages.SortFn)

	// Set root
	if root != nil {
		mqlPkg, err := newConanPackage(r.MqlRuntime, root)
		if err != nil {
			return err
		}
		r.Root = plugin.TValue[*mqlConanPackage]{Data: mqlPkg, State: plugin.StateIsSet}
	} else {
		r.Root = plugin.TValue[*mqlConanPackage]{State: plugin.StateIsSet | plugin.StateIsNull}
	}

	// Set list
	allResources, err := newConanPackageList(r.MqlRuntime, transitiveDeps)
	if err != nil {
		return err
	}
	r.List = plugin.TValue[[]any]{Data: allResources, State: plugin.StateIsSet}

	// Set direct
	directResources, err := newConanPackageList(r.MqlRuntime, directDeps)
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

func collectConanPackages(afs *afero.Afero, fs afero.Fs, path string) (*languages.Package, []*languages.Package, []*languages.Package, []string) {
	isDir, err := afs.IsDir(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("could not check Conan path")
		return nil, nil, nil, nil
	}

	if isDir {
		lockPath := filepath.Join(path, "conan.lock")
		if exists, _ := afs.Exists(lockPath); exists {
			return collectFromConanLock(afs, lockPath)
		}
		return nil, nil, nil, nil
	}

	if strings.HasSuffix(path, "conan.lock") {
		return collectFromConanLock(afs, path)
	}

	return nil, nil, nil, nil
}

func collectFromConanLock(afs *afero.Afero, path string) (*languages.Package, []*languages.Package, []*languages.Package, []string) {
	f, err := afs.Open(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("could not open conan.lock")
		return nil, nil, nil, nil
	}
	defer f.Close()

	extractor := &conanlock.Extractor{}
	bom, err := extractor.Parse(f, path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("could not parse conan.lock")
		return nil, nil, nil, nil
	}

	return bom.Root(), bom.Direct(), bom.Transitive(), []string{path}
}

func (r *mqlConanPackages) root() (*mqlConanPackage, error) {
	return nil, r.gatherData()
}

func (r *mqlConanPackages) directDependencies() ([]any, error) {
	return nil, r.gatherData()
}

func (r *mqlConanPackages) list() ([]any, error) {
	return nil, r.gatherData()
}

func (r *mqlConanPackages) files() ([]any, error) {
	return nil, r.gatherData()
}

func newConanPackageList(runtime *plugin.Runtime, packages []*languages.Package) ([]any, error) {
	resources := []any{}
	for i := range packages {
		pkg, err := newConanPackage(runtime, packages[i])
		if err != nil {
			return nil, err
		}
		resources = append(resources, pkg)
	}
	return resources, nil
}

func newConanPackage(runtime *plugin.Runtime, pkg *languages.Package) (*mqlConanPackage, error) {
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

	mqlPkg, err := CreateResource(runtime, "conan.package", map[string]*llx.RawData{
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
	return mqlPkg.(*mqlConanPackage), nil
}

func (k *mqlConanPackage) id() (string, error) {
	return k.Id.Data, nil
}

func (r *mqlConanPackage) name() (string, error) {
	return "", r.populateData()
}

func (r *mqlConanPackage) version() (string, error) {
	return "", r.populateData()
}

func (r *mqlConanPackage) purl() (string, error) {
	return "", r.populateData()
}

func (r *mqlConanPackage) cpes() ([]any, error) {
	return nil, r.populateData()
}

func (r *mqlConanPackage) files() ([]any, error) {
	return nil, r.populateData()
}

func (r *mqlConanPackage) populateData() error {
	return errors.New("conan.package can only be created via conan.packages")
}

func deduplicateConanPackages(pkgs []*languages.Package) []*languages.Package {
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
