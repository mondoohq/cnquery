package vsphere

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/license"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
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

func hostLockdownString(lockdownMode types.HostLockdownMode) string {
	var shortMode string
	shortMode = string(lockdownMode)
	shortMode = strings.ToLower(shortMode)
	shortMode = strings.TrimPrefix(shortMode, "lockdown")
	return shortMode
}
