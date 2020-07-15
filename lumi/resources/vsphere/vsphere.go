package vsphere

import (
	"context"
	"fmt"
	"net/url"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/license"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

type Config struct {
	User              string
	Password          string
	VSphereServerHost string
}

func (c *Config) vSphereURL() (*url.URL, error) {
	u, err := url.Parse("https://" + c.VSphereServerHost + "/sdk")
	if err != nil {
		return nil, err
	}
	u.User = url.UserPassword(c.User, c.Password)
	return u, nil
}

func New(cfg *Config) (*Client, error) {
	vsphereUrl, err := cfg.vSphereURL()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := govmomi.NewClient(ctx, vsphereUrl, true)
	if err != nil {
		return nil, err
	}

	instance := &Client{
		cfg:    cfg,
		client: client,
	}
	return instance, nil
}

type Client struct {
	cfg    *Config
	client *govmomi.Client
}

func (c *Client) ListLicenses() ([]types.LicenseManagerLicenseInfo, error) {
	manager := license.NewManager(c.client.Client)
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
	finder := find.NewFinder(c.client.Client, true)
	l, err := finder.ManagedObjectListChildren(context.TODO(), "/")
	if err != nil {
		return nil, nil
	}
	var dcs []*object.Datacenter
	for _, item := range l {
		if item.Object.Reference().Type == "Datacenter" {
			dc, err := getDatacenter(c.client, item.Path)
			if err != nil {
				return nil, err
			}
			dcs = append(dcs, dc)
		}
	}
	return dcs, nil
}

func getDatacenter(c *govmomi.Client, dc string) (*object.Datacenter, error) {
	finder := find.NewFinder(c.Client, true)
	t := c.ServiceContent.About.ApiType
	switch t {
	case "HostAgent":
		return finder.DefaultDatacenter(context.TODO())
	case "VirtualCenter":
		if dc != "" {
			return finder.Datacenter(context.TODO(), dc)
		}
		return finder.DefaultDatacenter(context.TODO())
	}
	return nil, fmt.Errorf("unsupported ApiType: %s", t)
}

func (c *Client) ListHosts() ([]*object.HostSystem, error) {
	finder := find.NewFinder(c.client.Client, true)
	return finder.HostSystemList(context.TODO(), "/DC0/host/DC0_C0/")
}

// func (c *Client) ListVirtualMachines() ([]*object.VirtualMachine, error) {
// 	finder := find.NewFinder(c.client.Client, true)
// 	return finder.VirtualMachineList(context.Background(), "/DC0/vm/DC0_C0/")
// }
