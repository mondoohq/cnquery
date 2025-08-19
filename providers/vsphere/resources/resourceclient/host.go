// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resourceclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/license"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
)

func HostInfo(host *object.HostSystem) (*mo.HostSystem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultAPITimeout)
	defer cancel()
	var props mo.HostSystem
	if err := host.Properties(ctx, host.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

func HostProperties(host *mo.HostSystem) (map[string]interface{}, error) {
	return PropertiesToDict(host)
}

func HostOptions(host *object.HostSystem) (map[string]interface{}, error) {
	ctx := context.Background()
	m, err := host.ConfigManager().OptionManager(ctx)
	if err != nil {
		return nil, err
	}

	var om mo.OptionManager
	err = m.Properties(ctx, m.Reference(), []string{"setting"}, &om)
	if err != nil {
		return nil, err
	}

	advancedProps := map[string]interface{}{}
	for i := range om.Setting {
		prop := om.Setting[i]
		key := prop.GetOptionValue().Key
		value := fmt.Sprintf("%v", prop.GetOptionValue().Value)
		advancedProps[key] = value
	}
	return advancedProps, nil
}

func HostServices(host *object.HostSystem) ([]types.HostService, error) {
	ctx := context.Background()
	ss, err := host.ConfigManager().ServiceSystem(ctx)
	if err != nil {
		return nil, err
	}
	return ss.Service(ctx)
}

func HostDateTime(host *object.HostSystem) (*types.HostDateTimeInfo, error) {
	ctx := context.Background()
	s, err := host.ConfigManager().DateTimeSystem(ctx)
	if err != nil {
		return nil, err
	}

	var hs mo.HostDateTimeSystem
	if err = s.Properties(ctx, s.Reference(), nil, &hs); err != nil {
		return nil, err
	}
	return &hs.DateTimeInfo, nil
}

func (c *Client) ListHosts(dc *object.Datacenter, cluster *object.ClusterComputeResource) ([]*object.HostSystem, error) {
	finder := find.NewFinder(c.Client.Client, true)

	// if we set a datacenter, use that as base path
	if dc != nil {
		finder.SetDatacenter(dc)
	}

	path := "*"

	// a cluster path will replace the  datacenter path, since it includes the datacenter
	if cluster != nil {
		path = cluster.InventoryPath + "/*"
	}

	res, err := finder.HostSystemList(context.Background(), path)
	if err != nil && IsNotFound(err) {
		return []*object.HostSystem{}, nil
	} else if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) HostByInventoryPath(path string) (*object.HostSystem, error) {
	finder := find.NewFinder(c.Client.Client, true)
	return finder.HostSystem(context.Background(), path)
}

func (c *Client) HostByMoid(moid types.ManagedObjectReference) (*object.HostSystem, error) {
	finder := find.NewFinder(c.Client.Client, true)
	ref, err := finder.ObjectReference(context.Background(), moid)
	if err != nil {
		return nil, err
	}

	switch ref.(type) {
	case *object.HostSystem:
		return ref.(*object.HostSystem), nil
	}
	return nil, errors.New("reference is not a valid host")
}

func HostLicenses(client *vim25.Client, hostID string) ([]types.LicenseManagerLicenseInfo, error) {
	ctx := context.Background()
	lm := license.NewManager(client)
	am, err := lm.AssignmentManager(ctx)
	if err != nil {
		return nil, err
	}

	assignedLicenses, err := am.QueryAssigned(ctx, hostID)
	if err != nil {
		return nil, err
	}

	res := make([]types.LicenseManagerLicenseInfo, len(assignedLicenses))
	for i := range assignedLicenses {
		res[i] = assignedLicenses[0].AssignedLicense
	}
	return res, nil
}

// GetHostTags retrieves tags for a host system using the vSphere vAPI
// If the vAPI is not available, it will return an empty array instead of an error.
// This maintains backward compatibility with vSphere environments that don't use tags.
func (c *Client) GetHostTags(ctx context.Context, hostRef types.ManagedObjectReference, conf *inventory.Config) []string {
	// Create vAPI REST client
	restClient := rest.NewClient(c.Client.Client)

	// Get credentials from connection config
	creds, err := vault.GetPassword(conf.Credentials)
	if err != nil {
		return []string{}
	}

	userInfo := url.UserPassword(creds.User, string(creds.Secret))
	err = restClient.Login(ctx, userInfo)
	if err != nil {
		return []string{}
	}

	tagManager := tags.NewManager(restClient)

	// Get attached tags for the host
	attachedTags, err := tagManager.GetAttachedTags(ctx, hostRef)
	if err != nil {
		return []string{}
	}

	// Convert tags to string format: "category:tag"
	tagStrings := make([]string, len(attachedTags))
	for i, tag := range attachedTags {
		// Get category information
		category, err := tagManager.GetCategory(ctx, tag.CategoryID)
		if err != nil {
			// If we can't get category, just use tag name
			tagStrings[i] = tag.Name
			continue
		}
		tagStrings[i] = fmt.Sprintf("%s:%s", category.Name, tag.Name)
	}

	return tagStrings
}

func hostLockdownString(lockdownMode types.HostLockdownMode) string {
	var shortMode string
	shortMode = string(lockdownMode)
	shortMode = strings.ToLower(shortMode)
	shortMode = strings.TrimPrefix(shortMode, "lockdown")
	return shortMode
}
