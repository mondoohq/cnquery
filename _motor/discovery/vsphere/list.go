package vsphere

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	provider "go.mondoo.com/cnquery/motor/providers/vsphere"
	"go.mondoo.com/cnquery/resources/packs/vsphere/resourceclient"
)

func New(client *govmomi.Client) *VSphere {
	return &VSphere{
		Client: client,
	}
}

type VSphere struct {
	Client *govmomi.Client
}

func (v *VSphere) InstanceUuid() (string, error) {
	return provider.InstanceUUID(v.Client)
}

func (v *VSphere) ListEsxiHosts() ([]*asset.Asset, error) {
	instanceUuid, err := v.InstanceUuid()
	if err != nil {
		return nil, err
	}

	dcs, err := v.listDatacenters()
	if err != nil {
		return nil, err
	}

	res := []*asset.Asset{}
	for i := range dcs {
		dc := dcs[i]
		hostList, err := v.listHosts(dc)
		if err != nil {
			return nil, err
		}
		hostsAsAssets, err := hostsToAssetList(instanceUuid, hostList)
		if err != nil {
			return nil, err
		}
		res = append(res, hostsAsAssets...)
	}
	return res, nil
}

func hostsToAssetList(instanceUuid string, hosts []*object.HostSystem) ([]*asset.Asset, error) {
	res := []*asset.Asset{}
	for i := range hosts {
		host := hosts[i]
		props, err := hostProperties(host)
		if err != nil {
			return nil, err
		}

		// NOTE: if a host is not running properly (returning not responding), the properties are nil
		ha := &asset.Asset{
			Name:  host.Name(),
			State: asset.State_STATE_UNKNOWN,
			Labels: map[string]string{
				"vsphere.vmware.com/name":          host.Name(),
				"vsphere.vmware.com/type":          host.Reference().Type,
				"vsphere.vmware.com/moid":          host.Reference().Encode(),
				"vsphere.vmware.com/inventorypath": host.InventoryPath,
			},
			PlatformIds: []string{provider.VsphereResourceID(instanceUuid, host.Reference())},
		}

		// add more information if available
		if props != nil && props.Config != nil {
			ha.Labels["vsphere.vmware.com/product-name"] = props.Config.Product.Name
			ha.Labels["vsphere.vmware.com/product-version"] = props.Config.Product.Version
			ha.Labels["vsphere.vmware.com/os-type"] = props.Config.Product.OsType
			ha.Labels["vsphere.vmware.com/produce-lineid"] = props.Config.Product.ProductLineId
			ha.State = mapHostPowerstateToState(props.Runtime.PowerState)
		}

		res = append(res, ha)
	}
	return res, nil
}

func hostProperties(host *object.HostSystem) (*mo.HostSystem, error) {
	ctx := context.Background()
	var props mo.HostSystem
	if err := host.Properties(ctx, host.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

func mapHostPowerstateToState(hostPowerState types.HostSystemPowerState) asset.State {
	switch hostPowerState {
	case types.HostSystemPowerStatePoweredOn:
		return asset.State_STATE_RUNNING
	case types.HostSystemPowerStatePoweredOff:
		return asset.State_STATE_STOPPED
	case types.HostSystemPowerStateStandBy:
		return asset.State_STATE_PENDING
	case types.HostSystemPowerStateUnknown:
		return asset.State_STATE_UNKNOWN
	default:
		return asset.State_STATE_UNKNOWN
	}
}

func (v *VSphere) ListVirtualMachines(parentTC *providers.Config) ([]*asset.Asset, error) {
	instanceUuid, err := v.InstanceUuid()
	if err != nil {
		return nil, err
	}

	dcs, err := v.listDatacenters()
	if err != nil {
		return nil, err
	}

	res := []*asset.Asset{}
	for i := range dcs {
		dc := dcs[i]
		vmList, err := v.listVirtualMachines(dc)
		if err != nil {
			return nil, err
		}
		vmsAsAssets, err := vmsToAssetList(instanceUuid, vmList, parentTC)
		if err != nil {
			return nil, err
		}
		res = append(res, vmsAsAssets...)
	}

	return res, nil
}

func vmsToAssetList(instanceUuid string, vms []*object.VirtualMachine, parentTC *providers.Config) ([]*asset.Asset, error) {
	res := []*asset.Asset{}
	for i := range vms {
		vm := vms[i]

		platformId := provider.VsphereResourceID(instanceUuid, vm.Reference())
		log.Debug().Str("platform-id", platformId).Msg("found vsphere vm")

		vmInfo, err := resourceclient.VmInfo(vm)
		if err != nil {
			return nil, err
		}

		guestState := mapVmGuestState(vmInfo.Guest.GuestState)

		ha := &asset.Asset{
			Name: vm.Name(),
			// TODO: derive platform information guest id e.g. debian10_64Guest, be aware that this does not need to be
			// the correct platform name
			State: guestState,
			Labels: map[string]string{
				"vsphere.vmware.com/name":           vm.Name(),
				"vsphere.vmware.com/type":           vm.Reference().Type,
				"vsphere.vmware.com/moid":           vm.Reference().Encode(),
				"vsphere.vmware.com/ip-address":     vmInfo.Guest.IpAddress,
				"vsphere.vmware.com/inventory-path": vm.InventoryPath,
				"vsphere.vmware.com/guest-hostname": vmInfo.Guest.HostName,
				"vsphere.vmware.com/guest-family":   vmInfo.Guest.GuestFamily,
				"vsphere.vmware.com/guest-id":       vmInfo.Guest.GuestId,
				"vsphere.vmware.com/guest-fullname": vmInfo.Guest.GuestFullName,
			},
			PlatformIds: []string{platformId},
		}

		if guestState == asset.State_STATE_RUNNING {
			ha.Connections = []*providers.Config{
				{
					Backend:     providers.ProviderType_VSPHERE_VM,
					Host:        parentTC.Host,
					Insecure:    parentTC.Insecure,
					Credentials: parentTC.Credentials,
					Options: map[string]string{
						"inventoryPath": vm.InventoryPath,
					},
				},
			}
		}

		res = append(res, ha)
	}
	return res, nil
}

func mapVmGuestState(vsphereGuestState string) asset.State {
	switch types.VirtualMachineGuestState(vsphereGuestState) {
	case types.VirtualMachineGuestStateRunning:
		return asset.State_STATE_RUNNING
	case types.VirtualMachineGuestStateShuttingDown:
		return asset.State_STATE_STOPPING
	case types.VirtualMachineGuestStateResetting:
		return asset.State_STATE_REBOOT
	case types.VirtualMachineGuestStateStandby:
		return asset.State_STATE_PENDING
	case types.VirtualMachineGuestStateNotRunning:
		return asset.State_STATE_STOPPED
	case types.VirtualMachineGuestStateUnknown:
		return asset.State_STATE_UNKNOWN
	default:
		return asset.State_STATE_UNKNOWN
	}
}

func (v *VSphere) listDatacenters() ([]*object.Datacenter, error) {
	finder := find.NewFinder(v.Client.Client, true)
	l, err := finder.ManagedObjectListChildren(context.Background(), "/")
	if err != nil {
		return nil, nil
	}
	var dcs []*object.Datacenter
	for _, item := range l {
		if item.Object.Reference().Type == "Datacenter" {
			dc, err := v.getDatacenter(item.Path)
			if err != nil {
				return nil, err
			}
			dcs = append(dcs, dc)
		}
	}
	return dcs, nil
}

func (v *VSphere) getDatacenter(dc string) (*object.Datacenter, error) {
	finder := find.NewFinder(v.Client.Client, true)
	t := v.Client.ServiceContent.About.ApiType
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

func (c *VSphere) listHosts(dc *object.Datacenter) ([]*object.HostSystem, error) {
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

func (c *VSphere) listVirtualMachines(dc *object.Datacenter) ([]*object.VirtualMachine, error) {
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
