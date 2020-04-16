package resources

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/mount"
)

func (p *lumiMount) id() (string, error) {
	return "mount", nil
}

func (s *lumiMount) GetList() ([]interface{}, error) {
	// find suitable mount manager
	mm, err := mount.ResolveManager(s.Runtime.Motor)
	if mm == nil || err != nil {
		return nil, fmt.Errorf("Could not detect suiteable mount manager for platform")
	}

	// retrieve all system packages
	osMounts, err := mm.List()
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve mount list for platform")
	}
	log.Debug().Int("mounts", len(osMounts)).Msg("lumi[mount]> mounted volumes")

	// create lumi mount entry resources for each mount
	mountEntries := make([]interface{}, len(osMounts))
	for i, osMount := range osMounts {

		// set init arguments for the lumi package resource
		args := make(lumi.Args)
		args["device"] = osMount.Device
		args["path"] = osMount.MountPoint
		args["fstype"] = osMount.FSType

		// convert options
		opts := map[string]interface{}{}
		for k := range osMount.Options {
			opts[k] = osMount.Options[k]
		}
		args["options"] = opts

		e, err := newMount_point(s.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("mount", osMount.Device).Msg("lumi[mount]> could not create mount entry resource")
			continue
		}
		mountEntries[i] = e.(Mount_point)
	}

	// return the mounts as new entries
	return mountEntries, nil
}

func (p *lumiMount_point) id() (string, error) {
	return p.Path()
}
