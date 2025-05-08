// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

func Detect(conn shared.Connection) ([]Package, map[string]PackageUpdate, error) {
	// find suitable package manager
	pms, err := ResolveSystemPkgManager(conn)
	if len(pms) == 0 || err != nil {
		return nil, nil, err
	}

	packages := []Package{}
	availableList := map[string]PackageUpdate{}
	for _, pm := range pms {
		// retrieve all system packages
		pkgs, err := pm.List()
		if err != nil {
			log.Debug().Err(err).Msg("mql[packages]> could not retrieve package list")
			return nil, nil, fmt.Errorf("could not retrieve package list for platform")
		}
		log.Debug().Int("packages", len(pkgs)).Msg("mql[packages]> installed packages")
		packages = append(packages, pkgs...)

		// TODO: do we really need to make this a blocking call, we could update available updates async
		// we try to retrieve the available updates
		available, err := pm.Available()
		if err != nil {
			log.Debug().Err(err).Msg("mql[packages]> could not retrieve available updates")
			available = map[string]PackageUpdate{}
		}
		log.Debug().Int("updates", len(available)).Msg("mql[packages]> available updates")
		for k, v := range available {
			availableList[k] = v
		}
	}

	return packages, availableList, nil
}

func ConvertParserPackages(pkgs []Package, updates map[string]PackageUpdate) []*mvd.Package {
	apiPkgs := []*mvd.Package{}

	for _, d := range pkgs {

		available := ""
		update, ok := updates[d.Name]
		if ok {
			available = update.Available
		}

		apiPkgs = append(apiPkgs, &mvd.Package{
			Name:      d.Name,
			Version:   d.Version,
			Available: available,
			Arch:      d.Arch,
			Origin:    d.Origin,
			Format:    d.Format,
		})
	}

	return apiPkgs
}
