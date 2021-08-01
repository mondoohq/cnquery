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
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports"
	vsphere_transport "go.mondoo.io/mondoo/motor/transports/vsphere"
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
	return vsphere_transport.InstanceUUID(v.Client)
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

		ha := &asset.Asset{
			Name: host.Name(),
			// NOTE: platform information is filled by the resolver
			State: mapHostPowerstateToState(props.Runtime.PowerState),
			Labels: map[string]string{
				"vsphere.vmware.com/name":            host.Name(),
				"vsphere.vmware.com/type":            host.Reference().Type,
				"vsphere.vmware.com/moid":            host.Reference().Encode(),
				"vsphere.vmware.com/inventorypath":   host.InventoryPath,
				"vsphere.vmware.com/product-name":    props.Config.Product.Name,
				"vsphere.vmware.com/product-version": props.Config.Product.Version,
				"vsphere.vmware.com/os-type":         props.Config.Product.OsType,
				"vsphere.vmware.com/produce-lineid":  props.Config.Product.ProductLineId,
			},
			PlatformIds: []string{vsphere_transport.VsphereResourceID(instanceUuid, host.Reference())},
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

func (v *VSphere) ListVirtualMachines(parentTC *transports.TransportConfig) ([]*asset.Asset, error) {
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

func vmsToAssetList(instanceUuid string, vms []*object.VirtualMachine, parentTC *transports.TransportConfig) ([]*asset.Asset, error) {
	res := []*asset.Asset{}
	for i := range vms {
		vm := vms[i]

		props, err := vmProperties(vm)
		if err != nil {
			return nil, err
		}

		platformId := vsphere_transport.VsphereResourceID(instanceUuid, vm.Reference())
		log.Debug().Str("platform-id", platformId).Msg("found vsphere vm")
		ha := &asset.Asset{
			Name: vm.Name(),
			// TODO: derive platform information guest id e.g. debian10_64Guest, be aware that this does not need to be
			// the correct platform name
			State: mapVmGuestState(props.Guest.GuestState),
			Labels: map[string]string{
				"vsphere.vmware.com/name":           vm.Name(),
				"vsphere.vmware.com/type":           vm.Reference().Type,
				"vsphere.vmware.com/moid":           vm.Reference().Encode(),
				"vsphere.vmware.com/ip-address":     props.Guest.IpAddress,
				"vsphere.vmware.com/inventory-path": vm.InventoryPath,
				"vsphere.vmware.com/guest-family":   props.Guest.GuestFamily,
				"vsphere.vmware.com/guest-id":       props.Guest.GuestId,
				"vsphere.vmware.com/guest-fullname": props.Guest.GuestFullName,
			},
			PlatformIds: []string{platformId},
		}

		// TODO: reactivate once we have multi-perspective scan active
		// add parent information to validate the vm configuration from vsphere api perspective
		// vt := parentTC.Clone()
		// ha.Connections = append(ha.Connections, vt)

		// TODO: steer the connection type by option
		ha.Connections = []*transports.TransportConfig{
			{
				Backend:     transports.TransportBackend_CONNECTION_VSPHERE_VM,
				Host:        parentTC.Host,
				Insecure:    parentTC.Insecure,
				Credentials: parentTC.Credentials,
				Options: map[string]string{
					"inventoryPath": vm.InventoryPath,
				},
			},
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

func vmProperties(vm *object.VirtualMachine) (*mo.VirtualMachine, error) {
	ctx := context.Background()
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
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
