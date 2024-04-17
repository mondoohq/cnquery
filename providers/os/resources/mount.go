// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/mount"
	"go.mondoo.com/cnquery/v11/types"
)

func (m *mqlMount) id() (string, error) {
	return "mount", nil
}

func (m *mqlMount) list() ([]interface{}, error) {
	// find suitable mount manager
	conn := m.MqlRuntime.Connection.(shared.Connection)
	mm, err := mount.ResolveManager(conn)
	if mm == nil || err != nil {
		return nil, fmt.Errorf("could not detect suitable mount manager for platform")
	}

	// retrieve all system packages
	osMounts, err := mm.List()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve mount list for platform")
	}
	log.Debug().Int("mounts", len(osMounts)).Msg("mql[mount]> mounted volumes")

	// create MQL mount entry resources for each mount
	mountEntries := make([]interface{}, len(osMounts))
	for i, osMount := range osMounts {
		// convert options
		opts := map[string]interface{}{}
		for k := range osMount.Options {
			opts[k] = osMount.Options[k]
		}

		o, err := CreateResource(m.MqlRuntime, "mount.point", map[string]*llx.RawData{
			"device":  llx.StringData(osMount.Device),
			"path":    llx.StringData(osMount.MountPoint),
			"fstype":  llx.StringData(osMount.FSType),
			"options": llx.MapData(opts, types.String),
			"mounted": llx.BoolTrue,
		})
		if err != nil {
			return nil, err
		}
		mountEntries[i] = o.(*mqlMountPoint)
	}

	// return the mounts as new entries
	return mountEntries, nil
}

func (m *mqlMountPoint) id() (string, error) {
	return m.Path.Data, nil
}

func initMountPoint(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	pathRaw := args["path"]
	if pathRaw == nil {
		return args, nil, nil
	}

	path, ok := pathRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := CreateResource(runtime, "mount", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	mount := obj.(*mqlMount)

	list := mount.GetList()
	if list.Error != nil {
		return nil, nil, list.Error
	}

	for i := range list.Data {
		mp := list.Data[i].(*mqlMountPoint)
		if mp.Path.Data == path {
			return nil, mp, nil
		}
	}

	return map[string]*llx.RawData{
		"device":  llx.StringData(""),
		"path":    llx.StringData(path),
		"fstype":  llx.StringData(""),
		"options": llx.MapData(nil, types.String),
		"mounted": llx.BoolFalse,
	}, nil, nil
}
