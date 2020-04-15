package resources

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/packages"
	motor "go.mondoo.io/mondoo/motor/motoros"
)

func (p *lumiOsupdate) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (p *lumiOsupdate) id() (string, error) {
	name, _ := p.Name()
	return name, nil
}

func (p *lumiOsUpdates) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (p *lumiOsUpdates) id() (string, error) {
	return "osupdates", nil
}

func (p *lumiOsUpdates) GetList() ([]interface{}, error) {
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
func (p *lumiOsUpdates) resolveOperatingSystemUpdateManager() (OperatingSystemUpdateManager, error) {
	var um OperatingSystemUpdateManager

	motor := p.Runtime.Motor
	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	// TODO: use OS family and select package manager
	switch platform.Name {
	case "opensuse": // suse family
		um = &SuseUpdateManager{motor: motor}
	default:
		return nil, errors.New("your platform is not supported by os updates resource")
	}
	return um, nil
}

type OperatingSystemUpdateManager interface {
	Name() string
	Format() string
	List() ([]packages.OperatingSystemUpdate, error)
}

type SuseUpdateManager struct {
	motor *motor.Motor
}

func (sum *SuseUpdateManager) Name() string {
	return "Suse Update Manager"
}

func (sum *SuseUpdateManager) Format() string {
	return "suse"
}

func (sum *SuseUpdateManager) List() ([]packages.OperatingSystemUpdate, error) {
	cmd, err := sum.motor.Transport.RunCommand("zypper --xmlout list-updates -t patch")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}
	return packages.ParseZypperPatches(cmd.Stdout)
}
