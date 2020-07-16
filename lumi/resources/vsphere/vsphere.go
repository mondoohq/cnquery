package vsphere

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
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
		Client: client,
	}
	return instance, nil
}

type Client struct {
	cfg    *Config
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

func hostProperties(host *object.HostSystem) (*mo.HostSystem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultAPITimeout)
	defer cancel()
	var props mo.HostSystem
	if err := host.Properties(ctx, host.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

func (c *Client) HostProperties(host *object.HostSystem) (map[string]interface{}, error) {
	props, err := hostProperties(host)
	if err != nil {
		return nil, err
	}

	dataProps := map[string]interface{}{}
	dataProps["PowerState"] = string(props.Runtime.PowerState)
	dataProps["ConnectionState"] = string(props.Runtime.ConnectionState)
	dataProps["InMaintenanceMode"] = strconv.FormatBool(props.Runtime.InMaintenanceMode)
	dataProps["LockdownMode"] = hostLockdownString(props.Config.LockdownMode)
	return dataProps, nil
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

// IsNotFound returns a boolean indicating whether the error is a not found error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var e *find.NotFoundError
	return errors.As(err, &e)
}
