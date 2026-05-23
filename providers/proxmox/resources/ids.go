// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"go.mondoo.com/mql/v13/providers/proxmox/connection"
)

// id() overrides for resources that need unique cache keys.
// Resources scoped to a parent (VM or Node) include the parent
// identifier to prevent cache collisions across VMs/nodes.

// --- Cluster-scoped ---

func (r *mqlProxmoxClusterHaResource) id() (string, error) {
	return "proxmox.cluster.haResource/" + r.Id.Data, nil
}

// --- Node-scoped (include node name to prevent multi-node collisions) ---

type mqlProxmoxNodeUpdateInternal struct {
	parentNode string
}

func (r *mqlProxmoxNodeUpdate) id() (string, error) {
	return fmt.Sprintf("proxmox.node.update/%s/%s", r.parentNode, r.Package.Data), nil
}

type mqlProxmoxNetworkInternal struct {
	parentNode string
}

func (r *mqlProxmoxNetwork) id() (string, error) {
	return fmt.Sprintf("proxmox.network/%s/%s", r.parentNode, r.Iface.Data), nil
}

type mqlProxmoxDnsInternal struct {
	parentNode string
}

func (r *mqlProxmoxDns) id() (string, error) {
	return fmt.Sprintf("proxmox.dns/%s", r.parentNode), nil
}

type mqlProxmoxServiceInternal struct {
	parentNode string
}

func (r *mqlProxmoxService) id() (string, error) {
	return fmt.Sprintf("proxmox.service/%s/%s", r.parentNode, r.Name.Data), nil
}

type mqlProxmoxCertificateInternal struct {
	parentNode string
}

func (r *mqlProxmoxCertificate) id() (string, error) {
	return fmt.Sprintf("proxmox.certificate/%s/%s", r.parentNode, r.Fingerprint.Data), nil
}

type mqlProxmoxSubscriptionInternal struct {
	parentNode string
}

func (r *mqlProxmoxSubscription) id() (string, error) {
	return fmt.Sprintf("proxmox.subscription/%s/%s", r.parentNode, r.ServerId.Data), nil
}

type mqlProxmoxRepositoryInternal struct {
	parentNode string
}

func (r *mqlProxmoxRepository) id() (string, error) {
	return fmt.Sprintf("proxmox.repository/%s/%s", r.parentNode, r.Id.Data), nil
}

// --- VM-scoped (include VMID to prevent cross-VM collisions) ---

type mqlProxmoxVmNetworkInternal struct {
	parentVmid int64
}

func (r *mqlProxmoxVmNetwork) id() (string, error) {
	return fmt.Sprintf("proxmox.vm.network/%d/%s", r.parentVmid, r.Id.Data), nil
}

type mqlProxmoxVmDiskInternal struct {
	parentVmid int64
}

func (r *mqlProxmoxVmDisk) id() (string, error) {
	return fmt.Sprintf("proxmox.vm.disk/%d/%s", r.parentVmid, r.Id.Data), nil
}

type mqlProxmoxVmSnapshotInternal struct {
	parentVmid int64
}

func (r *mqlProxmoxVmSnapshot) id() (string, error) {
	return fmt.Sprintf("proxmox.vm.snapshot/%d/%s", r.parentVmid, r.Name.Data), nil
}

type mqlProxmoxVmUpdateInternal struct {
	parentVmid int64
}

func (r *mqlProxmoxVmUpdate) id() (string, error) {
	return fmt.Sprintf("proxmox.vm.update/%d/%s", r.parentVmid, r.Name.Data), nil
}

type mqlProxmoxVmSerialPortInternal struct {
	parentVmid int64
}

func (r *mqlProxmoxVmSerialPort) id() (string, error) {
	return fmt.Sprintf("proxmox.vm.serialPort/%d/%s", r.parentVmid, r.Id.Data), nil
}

// --- Container-scoped (include VMID to prevent cross-container collisions) ---

type mqlProxmoxContainerNetworkInternal struct {
	parentVmid int64
}

func (r *mqlProxmoxContainerNetwork) id() (string, error) {
	return fmt.Sprintf("proxmox.container.network/%d/%s", r.parentVmid, r.Id.Data), nil
}

type mqlProxmoxContainerMountPointInternal struct {
	parentVmid int64
}

func (r *mqlProxmoxContainerMountPoint) id() (string, error) {
	return fmt.Sprintf("proxmox.container.mountPoint/%d/%s", r.parentVmid, r.Id.Data), nil
}

// --- Firewall rules (scoped to cluster/node/VM to prevent collisions) ---

type mqlProxmoxFirewallRuleInternal struct {
	scope string // e.g. "cluster", "node/pve1", "vm/100", "ct/200"
}

func (r *mqlProxmoxFirewallRule) id() (string, error) {
	return fmt.Sprintf("proxmox.firewall.rule/%s/%d/%s/%s/%s/%s",
		r.scope, r.Pos.Data, r.Type.Data, r.Action.Data, r.Source.Data, r.Dest.Data), nil
}

// --- Firewall options/ipsets/aliases (scoped) ---

type mqlProxmoxFirewallOptionsInternal struct{}

func (r *mqlProxmoxFirewallOptions) id() (string, error) {
	return "proxmox.firewall.options/" + r.Scope.Data, nil
}

type mqlProxmoxFirewallIpsetInternal struct {
	entriesScope string
	fetcherName  string
	fetcher      func(name string) ([]connection.IPSetEntry, error)
}

func (r *mqlProxmoxFirewallIpset) id() (string, error) {
	return "proxmox.firewall.ipset/" + r.Scope.Data + "/" + r.Name.Data, nil
}

type mqlProxmoxFirewallIpsetEntryInternal struct {
	scope   string // e.g. "cluster/myset", "vm/100/myset"
	ipsetID string
}

func (r *mqlProxmoxFirewallIpsetEntry) id() (string, error) {
	return "proxmox.firewall.ipset.entry/" + r.scope + "/" + r.Cidr.Data, nil
}

type mqlProxmoxFirewallAliasInternal struct{}

func (r *mqlProxmoxFirewallAlias) id() (string, error) {
	return "proxmox.firewall.alias/" + r.Scope.Data + "/" + r.Name.Data, nil
}

// --- Container (vmid is unique cluster-wide) ---

func (r *mqlProxmoxBackupJob) id() (string, error) {
	return "proxmox.backup.job/" + r.Id.Data, nil
}

// --- Globally unique (id field is already unique) ---

func (r *mqlProxmoxStorage) id() (string, error) {
	return "proxmox.storage/" + r.Id.Data, nil
}

func (r *mqlProxmoxPool) id() (string, error) {
	return "proxmox.pool/" + r.Id.Data, nil
}

func (r *mqlProxmoxUser) id() (string, error) {
	return "proxmox.user/" + r.Id.Data, nil
}

func (r *mqlProxmoxToken) id() (string, error) {
	return "proxmox.token/" + r.Id.Data, nil
}

func (r *mqlProxmoxRole) id() (string, error) {
	return "proxmox.role/" + r.Id.Data, nil
}

// --- HA group ---

func (r *mqlProxmoxClusterHaGroup) id() (string, error) {
	return "proxmox.cluster.haGroup/" + r.Id.Data, nil
}

// --- Node disks (node-scoped) ---

func (r *mqlProxmoxNodeDisk) id() (string, error) {
	return "proxmox.node.disk/" + r.parentNode + "/" + r.DevPath.Data, nil
}

type mqlProxmoxNodeDiskSmartInternal struct {
	parentNode string
	parentDev  string
}

func (r *mqlProxmoxNodeDiskSmart) id() (string, error) {
	return "proxmox.node.disk.smart/" + r.parentNode + "/" + r.parentDev, nil
}

func (r *mqlProxmoxZfsPool) id() (string, error) {
	return "proxmox.zfs.pool/" + r.parentNode + "/" + r.Name.Data, nil
}

func (r *mqlProxmoxLvmVolumeGroup) id() (string, error) {
	return "proxmox.lvm.volumeGroup/" + r.parentNode + "/" + r.Name.Data, nil
}

func (r *mqlProxmoxLvmThinPool) id() (string, error) {
	return "proxmox.lvm.thinPool/" + r.parentNode + "/" + r.VolumeGroup.Data + "/" + r.Name.Data, nil
}

// --- Replication ---

func (r *mqlProxmoxReplicationJob) id() (string, error) {
	return "proxmox.replication.job/" + r.Id.Data, nil
}

// --- SDN ---

func (r *mqlProxmoxSdnZone) id() (string, error) {
	return "proxmox.sdn.zone/" + r.Zone.Data, nil
}

func (r *mqlProxmoxSdnVnet) id() (string, error) {
	return "proxmox.sdn.vnet/" + r.Vnet.Data, nil
}

func (r *mqlProxmoxSdnSubnet) id() (string, error) {
	return "proxmox.sdn.subnet/" + r.Id.Data, nil
}
