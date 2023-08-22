// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"regexp"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/resources/packages"
	"go.mondoo.com/cnquery/utils/multierr"
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

	list := packages.GetList()
	if list.Error != nil {
		return nil, nil, list.Error
	}

	x, found := packages.packagesByName[name]
	if !found {
		return nil, nil, errors.New("cannot find package " + name)
	}

	return nil, x, nil
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
			// "epoch": "", // TODO: support Epoch
		})
		if err != nil {
			return nil, err
		}

		pkgs[i] = pkg
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
