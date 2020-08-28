package resources

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/resources/mount"
)

func (m *lumiMount) id() (string, error) {
	return "mount", nil
}

func (m *lumiMount) GetList() ([]interface{}, error) {
	// find suitable mount manager
	mm, err := mount.ResolveManager(m.Runtime.Motor)
	if mm == nil || err != nil {
		return nil, fmt.Errorf("could not detect suiteable mount manager for platform")
	}

	// retrieve all system packages
	osMounts, err := mm.List()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve mount list for platform")
	}
	log.Debug().Int("mounts", len(osMounts)).Msg("lumi[mount]> mounted volumes")

	// create lumi mount entry resources for each mount
	mountEntries := make([]interface{}, len(osMounts))
	for i, osMount := range osMounts {
		// convert options
		opts := map[string]interface{}{}
		for k := range osMount.Options {
			opts[k] = osMount.Options[k]
		}

		lumiMountEntry, err := m.Runtime.CreateResource("mount.point",
			"device", osMount.Device,
			"path", osMount.MountPoint,
			"fstype", osMount.FSType,
			"options", opts,
		)
		if err != nil {
			return nil, err
		}

		mountEntries[i] = lumiMountEntry.(MountPoint)
	}

	// return the mounts as new entries
	return mountEntries, nil
}

func (m *lumiMountPoint) id() (string, error) {
	return m.Path()
}
