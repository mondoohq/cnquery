package resolver

import (
	"context"

	"go.mondoo.io/mondoo/lumi/gql"

	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/resources/packages"
)

func (r *queryResolver) Packages(ctx context.Context) ([]gql.Package, error) {
	// find suitable package manager
	pm, err := packages.ResolveSystemPkgManager(r.Runtime.Motor)
	if pm == nil || err != nil {
		return nil, fmt.Errorf("Could not detect suiteable package manager for platform")
	}

	// retrieve all system packages
	osPkgs, err := pm.List()
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve package list for platform")
	}
	log.Debug().Int("packages", len(osPkgs)).Msg("lumi[packages]> installed packages")

	// create graphql packages
	pkgs := make([]gql.Package, len(osPkgs))
	for i, osPkg := range osPkgs {
		pkgs[i] = gql.Package{
			Name:        osPkg.Name,
			Version:     osPkg.Version,
			Arch:        osPkg.Arch,
			Status:      osPkg.Status,
			Description: osPkg.Description,
			Format:      pm.Format(),
		}
	}
	return pkgs, nil
}
