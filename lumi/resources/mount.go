package resources

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/mount"
)

func (m *lumiMount) id() (string, error) {
	return "mount", nil
}

func (m *lumiMount) GetList() ([]interface{}, error) {
	// find suitable mount manager
	mm, err := mount.ResolveManager(m.MotorRuntime.Motor)
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

		lumiMountEntry, err := m.MotorRuntime.CreateResource("mount.point",
			"device", osMount.Device,
			"path", osMount.MountPoint,
			"fstype", osMount.FSType,
			"options", opts,
			"mounted", true,
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

func (p *lumiMountPoint) init(args *lumi.Args) (*lumi.Args, MountPoint, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	pathRaw := (*args)["path"]
	if pathRaw == nil {
		return args, nil, nil
	}

	path, ok := pathRaw.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := p.MotorRuntime.CreateResource("mount")
	if err != nil {
		return nil, nil, err
	}
	mount := obj.(Mount)

	res, err := mount.List()
	if err != nil {
		return nil, nil, err
	}

	for i := range res {
		mp := res[i].(MountPoint)
		lumiMountPointPath, _ := mp.Path()
		if lumiMountPointPath == path {
			return nil, mp, nil
		}
	}

	// if the mount point cannot be found, we init it as an empty mount.point
	(*args)["device"] = ""
	(*args)["path"] = path
	(*args)["fstype"] = ""
	(*args)["options"] = map[string]interface{}{}
	(*args)["mounted"] = false

	return args, nil, nil
}
