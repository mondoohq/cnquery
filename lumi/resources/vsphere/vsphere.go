package vsphere

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/license"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25"
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
		return nil, nil
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

func (c *Client) ListClusters(dc *object.Datacenter) ([]*object.ClusterComputeResource, error) {
	finder := find.NewFinder(c.Client.Client, true)
	finder.SetDatacenter(dc)

	l, err := finder.ClusterComputeResourceList(context.Background(), "*")
	if err != nil && IsNotFound(err) {
		return []*object.ClusterComputeResource{}, nil
	} else if err != nil {
		return nil, err
	}
	return l, nil
}

func (c *Client) Cluster(path string) (*object.ClusterComputeResource, error) {
	finder := find.NewFinder(c.Client.Client, true)
	return finder.ClusterComputeResource(context.Background(), path)
}

func clusterProperties(cluster *object.ClusterComputeResource) (*mo.ClusterComputeResource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultAPITimeout)
	defer cancel()
	var props mo.ClusterComputeResource
	if err := cluster.Properties(ctx, cluster.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

func (c *Client) ClusterProperties(cluster *object.ClusterComputeResource) (map[string]interface{}, error) {
	props, err := clusterProperties(cluster)
	if err != nil {
		return nil, err
	}

	return PropertiesToDict(props)
}

func (c *Client) ListHosts(dc *object.Datacenter) ([]*object.HostSystem, error) {
	finder := find.NewFinder(c.Client.Client, true)
	finder.SetDatacenter(dc)
	res, err := finder.HostSystemList(context.Background(), "*")
	if err != nil && IsNotFound(err) {
		return []*object.HostSystem{}, nil
	} else if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) Host(path string) (*object.HostSystem, error) {
	finder := find.NewFinder(c.Client.Client, true)
	return finder.HostSystem(context.Background(), path)
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

func (c *Client) ListVirtualMachines(dc *object.Datacenter) ([]*object.VirtualMachine, error) {
	finder := find.NewFinder(c.Client.Client, true)
	finder.SetDatacenter(dc)
	res, err := finder.VirtualMachineList(context.Background(), "*")
	if err != nil && IsNotFound(err) {
		return []*object.VirtualMachine{}, nil
	} else if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) VirtualMachine(path string) (*object.VirtualMachine, error) {
	finder := find.NewFinder(c.Client.Client, true)
	return finder.VirtualMachine(context.Background(), path)
}

// IsNotFound returns a boolean indicating whether the error is a not found error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var e *find.NotFoundError
	return errors.As(err, &e)
}
