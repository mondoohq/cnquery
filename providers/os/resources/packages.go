// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"regexp"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/packages"
	"go.mondoo.com/cnquery/v11/types"
	"go.mondoo.com/cnquery/v11/utils/multierr"
)

var PKG_IDENTIFIER = regexp.MustCompile(`^(.*):\/\/(.*)\/(.*)\/(.*)$`)

// A system package cannot be installed twice but there are edge cases:
// - the same package name could be installed for multiple archs
// - linux-kernel package get extra treatment and can co-exist in multiple versions
// We use identifiers similar to grafeas artifact identifier for packages
// - deb://name/version/arch
// - rpm://name/version/arch
func (x *mqlPackage) id() (string, error) {
	return x.Format.Data + "://" + x.Name.Data + "/" + x.Version.Data + "/" + x.Arch.Data, nil
}

type mqlPackageInternal struct {
	filesState   packages.PkgFilesAvailable
	filesOnDisks []packages.FileRecord
}

// TODO: this is not accurate enough, we need to tie it to the package
func (x *mqlPkgFileInfo) id() (string, error) {
	return x.Path.Data, nil
}

func initPackage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// we only look up the package, if we have been supplied by its name and nothing else
	raw, ok := args["name"]
	if !ok || len(args) != 1 {
		return args, nil, nil
	}
	name := raw.Value.(string)

	pkgs, err := CreateResource(runtime, "packages", nil)
	if err != nil {
		return nil, nil, multierr.Wrap(err, "cannot get list of packages")
	}
	packages := pkgs.(*mqlPackages)

	if err = packages.refreshCache(nil); err != nil {
		return nil, nil, err
	}

	if res, ok := packages.packagesByName[name]; ok {
		return nil, res, nil
	}

	res := &mqlPackage{}
	res.Name = plugin.TValue[string]{Data: name, State: plugin.StateIsSet}
	res.Installed = plugin.TValue[bool]{Data: false, State: plugin.StateIsSet}
	res.Outdated = plugin.TValue[bool]{Data: false, State: plugin.StateIsSet}
	res.Version.State = plugin.StateIsSet | plugin.StateIsNull
	res.Epoch.State = plugin.StateIsSet | plugin.StateIsNull
	res.Available.State = plugin.StateIsSet | plugin.StateIsNull
	res.Description.State = plugin.StateIsSet | plugin.StateIsNull
	res.Purl.State = plugin.StateIsSet | plugin.StateIsNull
	res.Cpes.State = plugin.StateIsSet | plugin.StateIsNull
	res.Arch.State = plugin.StateIsSet | plugin.StateIsNull
	res.Format.State = plugin.StateIsSet | plugin.StateIsNull
	res.Origin.State = plugin.StateIsSet | plugin.StateIsNull
	res.Status.State = plugin.StateIsSet | plugin.StateIsNull
	res.Files.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, res, nil
}

func (p *mqlPackage) status() (string, error) {
	return "", nil
}

func (p *mqlPackage) outdated() (bool, error) {
	if len(p.Available.Data) > 0 {
		return true, nil
	}
	return false, nil
}

func (p *mqlPackage) origin() (string, error) {
	return "", nil
}

func (p *mqlPackage) files() ([]interface{}, error) {
	if p.filesState == packages.PkgFilesNotAvailable {
		return nil, nil
	}

	var filesOnDisk []packages.FileRecord

	if p.filesState == packages.PkgFilesIncluded {
		// we already have the data
		filesOnDisk = p.filesOnDisks
	} else {
		// we need to retrieve the data on-demand
		conn := p.MqlRuntime.Connection.(shared.Connection)
		pm, err := packages.ResolveSystemPkgManager(conn)
		if pm == nil || err != nil {
			return nil, errors.New("could not detect suitable package manager for platform")
		}
		filesOnDisk, err = pm.Files(p.Name.Data, p.Version.Data, p.Arch.Data)
		if err != nil {
			return nil, err
		}
	}

	var pkgFiles []interface{}
	for _, file := range filesOnDisk {
		pkgFile, err := CreateResource(p.MqlRuntime, "pkgFileInfo", map[string]*llx.RawData{
			"path": llx.StringData(file.Path),
		})
		if err != nil {
			return nil, err
		}
		pkgFiles = append(pkgFiles, pkgFile)
	}
	return pkgFiles, nil
}

type mqlPackagesInternal struct {
	lock           sync.Mutex
	packagesByName map[string]*mqlPackage
}

func (x *mqlPackages) list() ([]interface{}, error) {
	x.lock.Lock()
	defer x.lock.Unlock()

	conn := x.MqlRuntime.Connection.(shared.Connection)
	pm, err := packages.ResolveSystemPkgManager(conn)
	if pm == nil || err != nil {
		return nil, errors.New("could not detect suitable package manager for platform")
	}

	// retrieve all system packages
	osPkgs, err := pm.List()
	if err != nil {
		return nil, multierr.Wrap(err, "could not retrieve package list for platform")
	}

	// TODO: do we really need to make this a blocking call, we could update available updates async
	// we try to retrieve the available updates
	osAvailablePkgs, err := pm.Available()
	if err != nil {
		log.Debug().Err(err).Msg("mql[packages]> could not retrieve available updates")
		osAvailablePkgs = map[string]packages.PackageUpdate{}
	}

	// make available updates easily findable
	// we use packagename-arch as identifier
	availableMap := make(map[string]packages.PackageUpdate)
	for _, a := range osAvailablePkgs {
		availableMap[a.Name+"/"+a.Arch] = a
	}

	// create MQL package os for each package
	pkgs := make([]interface{}, len(osPkgs))
	for i, osPkg := range osPkgs {
		// check if we found a newer version
		available := ""
		update, ok := availableMap[osPkg.Name+"/"+osPkg.Arch]
		if ok {
			available = update.Available
			log.Debug().Str("package", osPkg.Name).Str("available", update.Available).Msg("mql[packages]> found newer version")
		}

		cpes := []interface{}{}
		if osPkg.CPE != "" {
			cpe, err := x.MqlRuntime.CreateSharedResource("cpe", map[string]*llx.RawData{
				"uri": llx.StringData(osPkg.CPE),
			})
			if err != nil {
				return nil, err
			}
			cpes = append(cpes, cpe)
		}

		pkg, err := CreateResource(x.MqlRuntime, "package", map[string]*llx.RawData{
			"name":        llx.StringData(osPkg.Name),
			"version":     llx.StringData(osPkg.Version),
			"available":   llx.StringData(available),
			"arch":        llx.StringData(osPkg.Arch),
			"status":      llx.StringData(osPkg.Status),
			"description": llx.StringData(osPkg.Description),
			"format":      llx.StringData(osPkg.Format),
			"installed":   llx.BoolData(true),
			"origin":      llx.StringData(osPkg.Origin),
			"epoch":       llx.StringData(osPkg.Epoch),
			"purl":        llx.StringData(osPkg.PUrl),
			"cpes":        llx.ArrayData(cpes, types.Resource("cpe")),
		})
		if err != nil {
			return nil, err
		}

		s := pkg.(*mqlPackage)
		s.filesState = osPkg.FilesAvailable
		s.filesOnDisks = osPkg.Files
		pkgs[i] = s
	}

	return pkgs, x.refreshCache(pkgs)
}

func (x *mqlPackages) refreshCache(all []interface{}) error {
	if all == nil {
		raw := x.GetList()
		if raw.Error != nil {
			return raw.Error
		}
		all = raw.Data
	}

	x.packagesByName = map[string]*mqlPackage{}

	for i := range all {
		u := all[i].(*mqlPackage)
		x.packagesByName[u.Name.Data] = u
	}

	return nil
}
