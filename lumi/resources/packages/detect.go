package packages

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/vadvisor/api"
)

func Detect(motor *motor.Motor) ([]Package, error) {
	// find suitable package manager
	pm, err := ResolveSystemPkgManager(motor)
	if pm == nil || err != nil {
		return nil, fmt.Errorf("could not detect suiteable package manager for platform")
	}

	// retrieve all system packages
	packages, err := pm.List()
	if err != nil {
		log.Debug().Err(err).Msg("lumi[packages]> could not retrieve package list")
		return nil, fmt.Errorf("could not retrieve package list for platform")
	}
	log.Debug().Int("packages", len(packages)).Msg("lumi[packages]> installed packages")

	// TODO: do we really need to make this a blocking call, we could update available updates async
	// we try to retrieve the available updates
	availableList, err := pm.Available()
	if err != nil {
		log.Debug().Err(err).Msg("lumi[packages]> could not retrieve available updates")
		availableList = []PackageUpdate{}
	}
	log.Debug().Int("updates", len(availableList)).Msg("lumi[packages]> available updates")

	return packages, nil
}

func ConvertPlatform(platform platform.Info) *api.Platform {
	return &api.Platform{
		Name:    platform.Name,
		Release: platform.Release,
		Arch:    platform.Arch,
	}
}

func ConvertParserPackages(pkgs []Package) []*api.Package {
	apiPkgs := []*api.Package{}

	for _, d := range pkgs {
		apiPkgs = append(apiPkgs, &api.Package{
			Name:    d.Name,
			Version: d.Version,
			Arch:    d.Arch,
			Origin:  d.Origin,
		})
	}

	return apiPkgs
}
