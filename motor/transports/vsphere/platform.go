package vsphere

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/govc/host/esxcli"
	"github.com/vmware/govmomi/object"
)

type EsxiSystemVersion struct {
	Build   string
	Patch   string
	Product string
	Update  string
	Version string
	Moid    string
}

func listDatacenters(c *govmomi.Client) ([]*object.Datacenter, error) {
	finder := find.NewFinder(c.Client, true)
	l, err := finder.ManagedObjectListChildren(context.Background(), "/")
	if err != nil {
		return nil, nil
	}
	var dcs []*object.Datacenter
	for _, item := range l {
		if item.Object.Reference().Type == "Datacenter" {
			dc, err := getDatacenter(c, item.Path)
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
		return finder.DefaultDatacenter(context.Background())
	case "VirtualCenter":
		if dc != "" {
			return finder.Datacenter(context.Background(), dc)
		}
		return finder.DefaultDatacenter(context.Background())
	}
	return nil, fmt.Errorf("unsupported ApiType: %s", t)
}

func listHosts(c *govmomi.Client, dc *object.Datacenter) ([]*object.HostSystem, error) {
	finder := find.NewFinder(c.Client, true)
	finder.SetDatacenter(dc)
	res, err := finder.HostSystemList(context.Background(), "*")
	if err != nil && IsNotFound(err) {
		return []*object.HostSystem{}, nil
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

// $ESXCli.system.version.get()
// Build   : Releasebuild-8169922
// Patch   : 0
// Product : VMware ESXi
// Update  : 0
// Version : 6.7.0
// see https://kb.vmware.com/s/article/2143832 for version and build number mapping
func (t *Transport) EsxiSystemVersion(host *object.HostSystem) (*EsxiSystemVersion, error) {
	e, err := esxcli.NewExecutor(t.client.Client, host)
	if err != nil {
		return nil, err
	}

	res, err := e.Run([]string{"system", "version", "get"})
	if err != nil {
		return nil, err
	}

	if len(res.Values) == 0 {
		return nil, errors.New("could not detect esxi system version ")
	}

	if len(res.Values) > 1 {
		return nil, errors.New("ambiguous esxi system version")
	}

	version := EsxiSystemVersion{}
	val := res.Values[0]
	for k := range val {
		if len(val[k]) == 1 {
			value := val[k][0]

			switch k {
			case "Build":
				version.Build = value
			case "Patch":
				version.Patch = value
			case "Product":
				version.Product = value
			case "Update":
				version.Update = value
			case "Version":
				version.Version = value
			}
		} else {
			log.Error().Str("key", k).Msg("system version> unsupported key")
		}
	}

	version.Moid = host.Reference().Value
	return &version, nil
}

func (t *Transport) GetHost() (*object.HostSystem, error) {
	dcs, err := listDatacenters(t.client)
	if err != nil {
		return nil, err
	}

	if len(dcs) != 1 {
		return nil, errors.New("esxi version only supported on esxi connection, found zero or multiple datacenters")
	}
	dc := dcs[0]

	hosts, err := listHosts(t.client, dc)
	if err != nil {
		return nil, err
	}

	if len(hosts) != 1 {
		return nil, errors.New("esxi version only supported on esxi connection, found zero or multiple hosts")
	}
	host := hosts[0]
	return host, nil
}

// TODO: handle case where we have a vsphere connection with multiple hosts
// in those cases we make the vsphere resources available, but return the esxi host
// version as the platform name
func (t *Transport) Identifier() ([]string, error) {
	identifier := []string{}

	// determine identifier
	if !t.Client().IsVC() {
		host, err := t.GetHost()
		if err != nil {
			return nil, err
		}

		// TODO: cache per connection
		version, err := t.EsxiSystemVersion(host)
		if err != nil {
			return nil, err
		}
		identifier = append(identifier, esxid(version.Moid))
	} else {
		v := t.Client().ServiceContent.About
		identifier = append(identifier, vsphereid(v.InstanceUuid))
	}
	return identifier, nil
}

func esxid(id string) string {
	return "//platformid.api.mondoo.app/runtime/esxi/moid/" + id
}

func vsphereid(id string) string {
	return "//platformid.api.mondoo.app/runtime/vsphere/uuid/" + id
}
