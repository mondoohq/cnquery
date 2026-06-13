// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resourceclient

import (
	"context"
	"fmt"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/license"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

const DefaultAPITimeout = time.Minute * 5

func New(client *govmomi.Client) *Client {
	return &Client{
		Client: client,
	}
}

type Client struct {
	Client *govmomi.Client
}

func (c *Client) AboutInfo() (map[string]any, error) {
	return PropertiesToDict(c.Client.ServiceContent.About)
}

// VcenterAdvancedSettings returns the vCenter Server advanced settings
// (the vpxd OptionManager exposed via PowerCLI as `Get-AdvancedSetting -Entity
// $VC`). This is distinct from per-host advanced settings, which come from each
// host's own OptionManager.
func (c *Client) VcenterAdvancedSettings(ctx context.Context) (map[string]string, error) {
	if c.Client.ServiceContent.Setting == nil {
		return map[string]string{}, nil
	}

	m := object.NewOptionManager(c.Client.Client, *c.Client.ServiceContent.Setting)
	var om mo.OptionManager
	if err := m.Properties(ctx, m.Reference(), []string{"setting"}, &om); err != nil {
		return nil, err
	}

	settings := map[string]string{}
	for i := range om.Setting {
		ov := om.Setting[i].GetOptionValue()
		settings[ov.Key] = fmt.Sprintf("%v", ov.Value)
	}
	return settings, nil
}

func (c *Client) ListLicenses() ([]types.LicenseManagerLicenseInfo, error) {
	manager := license.NewManager(c.Client.Client)
	infoList, err := manager.List(context.Background())
	if err != nil {
		return nil, err
	}

	res := []types.LicenseManagerLicenseInfo{}
	for _, info := range infoList {
		res = append(res, info)
	}
	return res, nil
}

func (c *Client) ListDatacenters() ([]*object.Datacenter, error) {
	finder := find.NewFinder(c.Client.Client, true)
	l, err := finder.ManagedObjectListChildren(context.Background(), "/")
	if err != nil {
		return nil, err
	}
	var dcs []*object.Datacenter
	for _, item := range l {
		if item.Object.Reference().Type == "Datacenter" {
			dc, err := getDatacenter(c.Client, item.Path)
			if err != nil {
				return nil, err
			}
			dcs = append(dcs, dc)
		}
	}
	return dcs, nil
}

func (c *Client) Datacenter(path string) (*object.Datacenter, error) {
	return getDatacenter(c.Client, path)
}

func getDatacenter(c *govmomi.Client, dc string) (*object.Datacenter, error) {
	finder := find.NewFinder(c.Client, true)
	t := c.ServiceContent.About.ApiType
	switch t {
	case "HostAgent":
		return finder.DefaultDatacenter(context.Background())
	case "VirtualCenter":
		if dc != "" {
			return finder.Datacenter(context.Background(), dc)
		}
		return finder.DefaultDatacenter(context.Background())
	}
	return nil, fmt.Errorf("unsupported ApiType: %s", t)
}
