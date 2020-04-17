package resources

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/packages"
	"go.mondoo.io/mondoo/lumi/resources/uptime"
)

func (p *lumiOs) id() (string, error) {
	return "os", nil
}

func (p *lumiOs) GetRebootpending() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (p *lumiOs) GetEnv() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (p *lumiOs) GetUptime() (int64, error) {
	uptime, err := uptime.New(p.Runtime.Motor)
	if err != nil {
		return 0, err
	}

	t, err := uptime.Duration()
	if err != nil {
		return 0, err
	}
	return int64(t), nil
}

// func (p *lumiOs) GetRebootpending() ([]interface{}, error) {
// 	return nil, errors.New("not implemented")
// }

func (p *lumiOsupdate) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (p *lumiOsupdate) id() (string, error) {
	name, _ := p.Name()
	return name, nil
}

func (p *lumiOs) GetUpdates() ([]interface{}, error) {
	// find suitable system updates
	um, err := p.resolveOperatingSystemUpdateManager()
	if um == nil || err != nil {
		return nil, fmt.Errorf("Could not detect suiteable update manager for platform")
	}

	// retrieve all system updates
	updates, err := um.List()
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve updates list for platform")
	}

	// create lumi update resources for each update
	osupdates := make([]interface{}, len(updates))
	log.Debug().Int("updates", len(updates)).Msg("lumi[updates]> found system updates")
	for i, update := range updates {

		// set init arguments for the lumi updates resource
		args := make(lumi.Args)
		args["name"] = update.Name
		args["severity"] = update.Severity
		args["category"] = update.Category
		args["restart"] = update.Restart
		args["format"] = um.Format()

		e, err := newOsupdate(p.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("update", update.Name).Msg("lumi[updates]> could not create osupdate resource")
			continue
		}
		osupdates[i] = e.(Osupdate)
	}

	// return the packages as new entries
	return osupdates, nil
}

// this will find the right package manager for the operating system
func (p *lumiOs) resolveOperatingSystemUpdateManager() (packages.OperatingSystemUpdateManager, error) {
	var um packages.OperatingSystemUpdateManager

	motor := p.Runtime.Motor
	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	// TODO: use OS family and select package manager
	switch platform.Name {
	case "opensuse": // suse family
		um = &packages.SuseUpdateManager{Motor: motor}
	default:
		return nil, errors.New("your platform is not supported by os updates resource")
	}
	return um, nil
}
