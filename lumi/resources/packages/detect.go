package packages

import (
	"fmt"

	"github.com/rs/zerolog/log"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/platform"
	"go.mondoo.io/mondoo/nexus/assets"
	"go.mondoo.io/mondoo/vadvisor/api"
)

func Detect(motor *motor.Motor) ([]Package, map[string]PackageUpdate, error) {
	// find suitable package manager
	pm, err := ResolveSystemPkgManager(motor)
	if pm == nil || err != nil {
		return nil, nil, err
	}

	// retrieve all system packages
	packages, err := pm.List()
	if err != nil {
		log.Debug().Err(err).Msg("lumi[packages]> could not retrieve package list")
		return nil, nil, fmt.Errorf("could not retrieve package list for platform")
	}
	log.Debug().Int("packages", len(packages)).Msg("lumi[packages]> installed packages")

	// TODO: do we really need to make this a blocking call, we could update available updates async
	// we try to retrieve the available updates
	availableList, err := pm.Available()
	if err != nil {
		log.Debug().Err(err).Msg("lumi[packages]> could not retrieve available updates")
		availableList = map[string]PackageUpdate{}
	}
	log.Debug().Int("updates", len(availableList)).Msg("lumi[packages]> available updates")

	return packages, availableList, nil
}

func ConvertPlatform(platform platform.Info) *assets.Platform {
	return &assets.Platform{
		Name:    platform.Name,
		Release: platform.Release,
		Arch:    platform.Arch,
	}
}

func ConvertParserPackages(pkgs []Package, updates map[string]PackageUpdate) []*api.Package {
	apiPkgs := []*api.Package{}

	for _, d := range pkgs {

		available := ""
		update, ok := updates[d.Name]
		if ok {
			available = update.Available
		}

		apiPkgs = append(apiPkgs, &api.Package{
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
