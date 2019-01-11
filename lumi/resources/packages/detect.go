package packages

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/resources/parser"
	"go.mondoo.io/mondoo/motor"
)

func Detect(motor *motor.Motor) ([]parser.Package, error) {
	// find suitable package manager
	pm, err := ResolveSystemPkgManager(motor)
	if pm == nil || err != nil {
		return nil, fmt.Errorf("Could not detect suiteable package manager for platform")
	}

	// retrieve all system packages
	packages, err := pm.List()
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve package list for platform")
	}
	log.Debug().Int("packages", len(packages)).Msg("lumi[packages]> installed packages")

	// TODO: do we really need to make this a blocking call, we could update available updates async
	// we try to retrieve the available updates
	availableList, err := pm.Available()
	if err != nil {
		log.Debug().Err(err).Msg("lumi[packages]> could not retrieve available updates")
		availableList = []parser.PackageUpdate{}
	}
	log.Debug().Int("updates", len(availableList)).Msg("lumi[packages]> available updates")

	return packages, nil
}
