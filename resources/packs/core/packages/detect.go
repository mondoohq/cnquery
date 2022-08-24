package packages

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/resources/packs/core/vadvisor"
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
		log.Debug().Err(err).Msg("mql[packages]> could not retrieve package list")
		return nil, nil, fmt.Errorf("could not retrieve package list for platform")
	}
	log.Debug().Int("packages", len(packages)).Msg("mql[packages]> installed packages")

	// TODO: do we really need to make this a blocking call, we could update available updates async
	// we try to retrieve the available updates
	availableList, err := pm.Available()
	if err != nil {
		log.Debug().Err(err).Msg("mql[packages]> could not retrieve available updates")
		availableList = map[string]PackageUpdate{}
	}
	log.Debug().Int("updates", len(availableList)).Msg("mql[packages]> available updates")

	return packages, availableList, nil
}

func ConvertParserPackages(pkgs []Package, updates map[string]PackageUpdate) []*vadvisor.Package {
	apiPkgs := []*vadvisor.Package{}

	for _, d := range pkgs {

		available := ""
		update, ok := updates[d.Name]
		if ok {
			available = update.Available
		}

		apiPkgs = append(apiPkgs, &vadvisor.Package{
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
