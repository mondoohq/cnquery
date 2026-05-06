// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"

	"github.com/vmware/govmomi/vim25/types"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// vsphereInventory indexes hosts, VMs, datastores, and clusters by encoded
// moid for cross-reference resolution. Built once per scan and cached on the
// singleton vsphere resource (see mqlVsphereInternal), since a query like
// `vsphere.vms { host datastores }` over N VMs would otherwise rebuild the
// index 2N times.
type vsphereInventory struct {
	hosts      map[string]*mqlVsphereHost
	vms        map[string]*mqlVsphereVm
	datastores map[string]*mqlVsphereDatastore
	clusters   map[string]*mqlVsphereCluster
}

// mqlVsphereInternal extends mqlVspherePermissionInternal (already declared in
// vsphere.go); the codegen embeds both fields into the generated resource
// struct. The sync.Once ensures the inventory index is built exactly once per
// scan even when called concurrently from multiple cross-reference accessors.
type mqlVsphereInternal struct {
	inventoryOnce sync.Once
	inventory     *vsphereInventory
	inventoryErr  error
}

func loadVsphereInventory(runtime *plugin.Runtime) (*vsphereInventory, error) {
	res, err := CreateResource(runtime, "vsphere", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	v := res.(*mqlVsphere)
	v.inventoryOnce.Do(func() {
		v.inventory, v.inventoryErr = buildVsphereInventory(v)
	})
	return v.inventory, v.inventoryErr
}

func buildVsphereInventory(v *mqlVsphere) (*vsphereInventory, error) {
	dcs := v.GetDatacenters()
	if dcs.Error != nil {
		return nil, dcs.Error
	}
	inv := &vsphereInventory{
		hosts:      map[string]*mqlVsphereHost{},
		vms:        map[string]*mqlVsphereVm{},
		datastores: map[string]*mqlVsphereDatastore{},
		clusters:   map[string]*mqlVsphereCluster{},
	}
	for _, d := range dcs.Data {
		dc := d.(*mqlVsphereDatacenter)
		if hosts := dc.GetHosts(); hosts.Error == nil {
			for _, h := range hosts.Data {
				host := h.(*mqlVsphereHost)
				inv.hosts[host.Moid.Data] = host
			}
		}
		if vms := dc.GetVms(); vms.Error == nil {
			for _, vm := range vms.Data {
				m := vm.(*mqlVsphereVm)
				inv.vms[m.Moid.Data] = m
			}
		}
		if datastores := dc.GetDatastores(); datastores.Error == nil {
			for _, ds := range datastores.Data {
				s := ds.(*mqlVsphereDatastore)
				inv.datastores[s.Moid.Data] = s
			}
		}
		if clusters := dc.GetClusters(); clusters.Error == nil {
			for _, c := range clusters.Data {
				cl := c.(*mqlVsphereCluster)
				inv.clusters[cl.Moid.Data] = cl
			}
		}
	}
	return inv, nil
}

// resolveHosts maps a slice of HostSystem refs to their MQL resources, dropping
// refs that aren't in inventory (e.g. hosts the caller doesn't have read
// access to).
func (inv *vsphereInventory) resolveHosts(refs []types.ManagedObjectReference) []any {
	out := make([]any, 0, len(refs))
	for _, r := range refs {
		if h, ok := inv.hosts[r.Encode()]; ok {
			out = append(out, h)
		}
	}
	return out
}

func (inv *vsphereInventory) resolveVms(refs []types.ManagedObjectReference) []any {
	out := make([]any, 0, len(refs))
	for _, r := range refs {
		if v, ok := inv.vms[r.Encode()]; ok {
			out = append(out, v)
		}
	}
	return out
}

func (inv *vsphereInventory) resolveDatastores(refs []types.ManagedObjectReference) []any {
	out := make([]any, 0, len(refs))
	for _, r := range refs {
		if d, ok := inv.datastores[r.Encode()]; ok {
			out = append(out, d)
		}
	}
	return out
}

// host resolves the ESXi host the VM is currently running on against
// vsphere.datacenters[].hosts. Null when the VM is unregistered or the host
// isn't visible in inventory.
func (v *mqlVsphereVm) host() (*mqlVsphereHost, error) {
	if v.vm == nil || v.vm.Runtime.Host == nil {
		v.Host.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	inv, err := loadVsphereInventory(v.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if h, ok := inv.hosts[v.vm.Runtime.Host.Encode()]; ok {
		return h, nil
	}
	v.Host.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

// datastores resolves the VM's backing datastores against
// vsphere.datacenters[].datastores.
func (v *mqlVsphereVm) datastores() ([]any, error) {
	if v.vm == nil {
		return []any{}, nil
	}
	inv, err := loadVsphereInventory(v.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return inv.resolveDatastores(v.vm.Datastore), nil
}

// cluster resolves the host's parent cluster against
// vsphere.datacenters[].clusters. Null for standalone hosts (whose parent is a
// ComputeResource, not a ClusterComputeResource).
func (v *mqlVsphereHost) cluster() (*mqlVsphereCluster, error) {
	if v.host == nil || v.host.Parent == nil || v.host.Parent.Type != "ClusterComputeResource" {
		v.Cluster.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	inv, err := loadVsphereInventory(v.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if c, ok := inv.clusters[v.host.Parent.Encode()]; ok {
		return c, nil
	}
	v.Cluster.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

// datastores resolves the host's mounted datastores against
// vsphere.datacenters[].datastores.
func (v *mqlVsphereHost) datastores() ([]any, error) {
	if v.host == nil {
		return []any{}, nil
	}
	inv, err := loadVsphereInventory(v.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return inv.resolveDatastores(v.host.Datastore), nil
}

// hosts resolves the hosts that have this datastore mounted against
// vsphere.datacenters[].hosts. Datastore.Host carries DatastoreHostMount; we
// use its Key (the host moid).
func (v *mqlVsphereDatastore) hosts() ([]any, error) {
	if v.ds == nil {
		return []any{}, nil
	}
	refs := make([]types.ManagedObjectReference, 0, len(v.ds.Host))
	for _, h := range v.ds.Host {
		refs = append(refs, h.Key)
	}
	inv, err := loadVsphereInventory(v.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return inv.resolveHosts(refs), nil
}

// vms resolves the VMs whose files reside on this datastore against
// vsphere.datacenters[].vms.
func (v *mqlVsphereDatastore) vms() ([]any, error) {
	if v.ds == nil {
		return []any{}, nil
	}
	inv, err := loadVsphereInventory(v.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return inv.resolveVms(v.ds.Vm), nil
}
