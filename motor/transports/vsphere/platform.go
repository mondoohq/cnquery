package vsphere

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/govc/host/esxcli"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"go.mondoo.io/mondoo/nexus/mrn"
)

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
func (t *Transport) EsxiVersion(host *object.HostSystem) (*EsxiSystemVersion, error) {
	return EsxiVersion(host)
}

type EsxiSystemVersion struct {
	Build   string
	Patch   string
	Product string
	Update  string
	Version string
	Moid    string
}

func EsxiVersion(host *object.HostSystem) (*EsxiSystemVersion, error) {
	e, err := esxcli.NewExecutor(host.Client(), host)
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

// Identifier will only identify the connection
// see https://blogs.vmware.com/vsphere/2012/02/uniquely-identifying-virtual-machines-in-vsphere-and-vcloud-part-1-overview.html
// To match the vm with the guest, we would need to extract the vm uuid from bios
// https://kb.vmware.com/s/article/1009458
// /usr/sbin/dmidecode | grep UUID https://communities.vmware.com/thread/420420
// wmic bios get name,serialnumber,version  https://communities.vmware.com/thread/582729/
func (t *Transport) Identifier() (string, error) {
	// a specific resource id was passed into the transport eg. for a esxi host or esxi vm
	if len(t.resid) > 0 {
		return t.resid, nil
	}

	// determine identifier since ESXI connections do not return an InstanceUuid
	if !t.Client().IsVC() {
		host, err := t.GetHost()
		if err != nil {
			return "", err
		}

		// NOTE: we do not use the ESXi host identifier here to distingush between the API and the host itself
		return VsphereID(host.Reference().Value), nil
	}

	v := t.Client().ServiceContent.About
	return VsphereID(v.InstanceUuid), nil
}

// Info returns the connection information
func (t *Transport) Info() types.AboutInfo {
	return t.Client().ServiceContent.About
}

func VsphereResourceID(typ string, inventorypath string) string {
	return "//platformid.api.mondoo.app/runtime/vsphere/type/" + typ + "/inventorypath/" + base64.StdEncoding.EncodeToString([]byte(inventorypath))
}

func ParseVsphereResourceID(id string) (string, string, error) {
	parsed, err := mrn.NewMRN(id)
	if err != nil {
		return "", "", err
	}

	typ := parsed.ResourceID("type")
	if typ == nil {
		return "", "", errors.New("vsphere platform id has invalid type")
	}
	inventoryPath := parsed.ResourceID("inventorypath")

	var decodedPath []byte
	if inventoryPath != nil {
		base64path := *inventoryPath
		decodedPath, err = base64.StdEncoding.DecodeString(base64path)
		if err != nil {
			return "", "", errors.New("vsphere platform id has invalid inventorypath")
		}
	}

	return *typ, string(decodedPath), nil

}

func IsVsphereResourceID(mrn string) bool {
	return strings.HasPrefix(mrn, "//platformid.api.mondoo.app/runtime/vsphere/type/") && strings.Contains(mrn, "/inventorypath/")
}

// use in combination with Client.ServiceContent.About.InstanceUuid
func VsphereID(id string) string {
	return "//platformid.api.mondoo.app/runtime/vsphere/uuid/" + id
}

func IsVsphereID(mrn string) bool {
	return strings.HasPrefix(mrn, "//platformid.api.mondoo.app/runtime/vsphere/uuid/")
}

func (c *Transport) Host(path string) (*object.HostSystem, error) {
	finder := find.NewFinder(c.Client().Client, true)
	return finder.HostSystem(context.Background(), path)
}

func (c *Transport) VirtualMachine(path string) (*object.VirtualMachine, error) {
	finder := find.NewFinder(c.Client().Client, true)
	return finder.VirtualMachine(context.Background(), path)
}
