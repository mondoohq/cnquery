// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resourceclient

import (
	"context"
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
)

func (c *Client) GetDistributedVirtualSwitches(ctx context.Context, path string) ([]*object.DistributedVirtualSwitch, error) {
	finder := find.NewFinder(c.Client.Client, true)
	mos, err := finder.ManagedObjectListChildren(ctx, path)
	if err != nil {
		return nil, err
	}
	if len(mos) == 0 {
		return nil, nil
	}

	var dvSwitches []*object.DistributedVirtualSwitch
	for _, item := range mos {
		if !strings.HasSuffix(item.Path, "/network") {
			continue
		}
		log.Debug().Str("object", item.String()).Msg("???")
		ref := item.Object.Reference()
		log.Debug().Str("ref", ref.Type).Msg("???")
		networkMos, err := finder.ManagedObjectListChildren(ctx, item.Path)
		if err != nil {
			return nil, err
		}
		for _, networkMo := range networkMos {
			log.Debug().Str("networkMo", networkMo.String()).Msg("???")
			if networkMo.Object.Reference().Type == "DistributedVirtualSwitch" || networkMo.Object.Reference().Type == "VmwareDistributedVirtualSwitch" {
				dvpg, err := finder.Network(ctx, networkMo.Path)
				if err != nil {
					return nil, err
				}
				casted, ok := dvpg.(*object.DistributedVirtualSwitch)
				if !ok {
					return nil, errors.New("not a DistributedVirtualSwitch reference")
				}
				dvSwitches = append(dvSwitches, casted)
			}
		}
	}

	return dvSwitches, nil
}

func (c *Client) GetDistributedVirtualSwitchConfig(ctx context.Context, obj *object.DistributedVirtualSwitch) (*mo.DistributedVirtualSwitch, error) {
	var moDvSwitch mo.DistributedVirtualSwitch
	err := obj.Properties(ctx, obj.Reference(), nil, &moDvSwitch)
	if err != nil {
		return nil, err
	}

	return &moDvSwitch, nil
}

func DistributedVirtualSwitchConfig(dvSwitch *mo.DistributedVirtualSwitch) (map[string]interface{}, error) {
	return PropertiesToDict(dvSwitch)
}

func (c *Client) GetDistributedVirtualPortgroups(ctx context.Context, path string) ([]*object.DistributedVirtualPortgroup, error) {
	finder := find.NewFinder(c.Client.Client, true)
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 2 {
		return nil, nil
	}
	dcPath := "/" + pathParts[1]
	dc, err := c.Datacenter(dcPath)
	if err != nil {
		return nil, err
	}
	finder.SetDatacenter(dc)

	mos, err := finder.ManagedObjectListChildren(ctx, dcPath)
	if err != nil {
		return nil, err
	}
	if len(mos) == 0 {
		return nil, nil
	}

	var pgs []*object.DistributedVirtualPortgroup
	for _, item := range mos {
		if !strings.HasSuffix(item.Path, "/network") {
			continue
		}
		log.Debug().Str("object", item.String()).Msg("???")
		ref := item.Object.Reference()
		log.Debug().Str("ref", ref.Type).Msg("???")
		networkMos, err := finder.ManagedObjectListChildren(ctx, item.Path)
		if err != nil {
			return nil, err
		}
		for _, networkMo := range networkMos {
			log.Debug().Str("networkMo", networkMo.String()).Msg("???")
			if networkMo.Object.Reference().Type == "DistributedVirtualPortgroup" {
				dvpg, err := finder.Network(ctx, networkMo.Path)
				if err != nil {
					return nil, err
				}
				casted, ok := dvpg.(*object.DistributedVirtualPortgroup)
				if !ok {
					return nil, errors.New("not a DistributedVirtualPortgroup reference")
				}

				pgs = append(pgs, casted)
			}
		}
	}

	return pgs, nil
}

func (c *Client) GetDistributedVirtualPortgroupConfig(ctx context.Context, obj *object.DistributedVirtualPortgroup) (*mo.DistributedVirtualPortgroup, error) {
	var moDvpg mo.DistributedVirtualPortgroup
	err := obj.Properties(ctx, obj.Reference(), nil, &moDvpg)
	if err != nil {
		return nil, err
	}

	return &moDvpg, nil
}

func DistributedVirtualPortgroupConfig(dvpg *mo.DistributedVirtualPortgroup) (map[string]interface{}, error) {
	return PropertiesToDict(dvpg)
}
