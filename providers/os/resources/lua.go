// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"path"
	"slices"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/lua/luarocks"
	"go.mondoo.com/mql/v13/types"
)

// Default paths where LuaRocks installs packages
var defaultLuaRocksPaths = []string{
	"/usr/local/lib/luarocks/rocks-5.4",
	"/usr/local/lib/luarocks/rocks-5.3",
	"/usr/local/lib/luarocks/rocks-5.1",
	"/usr/share/lua/5.4",
	"/usr/share/lua/5.3",
	"/usr/share/lua/5.1",
}

func initLuaPackages(_ *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		_, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in lua.packages initialization, it must be a string")
		}
	} else {
		args["path"] = llx.StringData("")
	}
	return args, nil, nil
}

func (r *mqlLuaPackages) id() (string, error) {
	if r.Path.Data != "" {
		return "lua.packages/" + r.Path.Data, nil
	}
	return "lua.packages", nil
}

type mqlLuaPackagesInternal struct {
	mutex   sync.Mutex
	fetched bool
}

func (r *mqlLuaPackages) gatherData() error {
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

	searchPath := r.Path.Data

	var transitiveDeps []*languages.Package
	var filePaths []string

	if searchPath != "" {
		t, f := collectLuaPackages(afs, searchPath)
		transitiveDeps = append(transitiveDeps, t...)
		filePaths = append(filePaths, f...)
	} else {
		// Try CLI first
		if conn.Capabilities().Has(shared.Capability_RunCommand) {
			cmd, err := conn.RunCommand("luarocks list --porcelain")
			if err == nil && cmd.ExitStatus == 0 {
				pkgs, fps := luarocks.ParseLuaRocksList(cmd.Stdout, "")
				transitiveDeps = append(transitiveDeps, pkgs...)
				filePaths = append(filePaths, fps...)
			} else {
				exitStatus := -1
				if cmd != nil {
					exitStatus = cmd.ExitStatus
				}
				log.Debug().Err(err).Int("exitStatus", exitStatus).Msg("mql[lua]> luarocks list failed, falling back to filesystem")
			}
		}

		// If CLI didn't return results, scan filesystem
		if len(transitiveDeps) == 0 {
			for _, rocksPath := range defaultLuaRocksPaths {
				pkgs, fps := luarocks.ParseRocksDir(afs, rocksPath)
				transitiveDeps = append(transitiveDeps, pkgs...)
				filePaths = append(filePaths, fps...)
			}
		}
	}

	slices.SortFunc(transitiveDeps, languages.SortFn)

	allResources, err := newLuaPackageList(r.MqlRuntime, transitiveDeps)
	if err != nil {
		return err
	}
	r.List = plugin.TValue[[]any]{Data: allResources, State: plugin.StateIsSet}

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

func collectLuaPackages(afs *afero.Afero, searchPath string) ([]*languages.Package, []string) {
	isDir, err := afs.IsDir(searchPath)
	if err != nil {
		return nil, nil
	}

	if isDir {
		// Check if it's a rocks directory directly
		pkgs, fps := luarocks.ParseRocksDir(afs, searchPath)
		if len(pkgs) > 0 {
			return pkgs, fps
		}

		// Accumulate results across all rocks subdirectories
		var allPkgs []*languages.Package
		var allFps []string
		for _, suffix := range []string{"rocks-5.4", "rocks-5.3", "rocks-5.1"} {
			// path.Join (not filepath.Join) — always Linux paths
			rocksPath := path.Join(searchPath, suffix)
			pkgs, fps := luarocks.ParseRocksDir(afs, rocksPath)
			allPkgs = append(allPkgs, pkgs...)
			allFps = append(allFps, fps...)
		}
		return allPkgs, allFps
	}

	return nil, nil
}

func (r *mqlLuaPackages) list() ([]any, error) {
	return nil, r.gatherData()
}

func (r *mqlLuaPackages) files() ([]any, error) {
	return nil, r.gatherData()
}

func newLuaPackageList(runtime *plugin.Runtime, packages []*languages.Package) ([]any, error) {
	resources := []any{}
	for i := range packages {
		pkg, err := newLuaPackage(runtime, packages[i])
		if err != nil {
			return nil, err
		}
		resources = append(resources, pkg)
	}
	return resources, nil
}

func newLuaPackage(runtime *plugin.Runtime, pkg *languages.Package) (*mqlLuaPackage, error) {
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

	mqlPkg, err := CreateResource(runtime, "lua.package", map[string]*llx.RawData{
		"id":      llx.StringData(pkg.Name + "@" + pkg.Version + ":" + path),
		"name":    llx.StringData(pkg.Name),
		"version": llx.StringData(pkg.Version),
		"purl":    llx.StringData(pkg.Purl),
		"files":   llx.ArrayData(mqlFiles, types.Resource("pkgFileInfo")),
	})
	if err != nil {
		return nil, err
	}
	return mqlPkg.(*mqlLuaPackage), nil
}

func (k *mqlLuaPackage) id() (string, error) {
	return k.Id.Data, nil
}

func (r *mqlLuaPackage) name() (string, error) {
	return "", r.populateData()
}

func (r *mqlLuaPackage) version() (string, error) {
	return "", r.populateData()
}

func (r *mqlLuaPackage) purl() (string, error) {
	return "", r.populateData()
}

func (r *mqlLuaPackage) files() ([]any, error) {
	return nil, r.populateData()
}

func (r *mqlLuaPackage) populateData() error {
	return errors.New("lua.package can only be created via lua.packages")
}
