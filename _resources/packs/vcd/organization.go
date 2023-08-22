// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package vcd

import (
	"strconv"

	"go.mondoo.com/cnquery/resources/packs/core"

	"github.com/rs/zerolog/log"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"go.mondoo.com/cnquery/resources"
)

func (v *mqlVcd) GetOrganizations() ([]interface{}, error) {
	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	orgs, err := client.GetOrgList()
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range orgs.Org {
		entry, err := newMqlVcdOrganization(v.MotorRuntime, orgs.Org[i])
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}
	return list, nil
}

func newMqlVcdOrganization(runtime *resources.Runtime, org *types.Org) (interface{}, error) {
	return runtime.CreateResource("vcd.organization",
		"id", org.ID,
		"name", org.Name,
		"fullName", org.FullName,
		"isEnabled", org.IsEnabled,
		"description", org.Description,
	)
}

func (v *mqlVcdOrganization) id() (string, error) {
	id, err := v.Name()
	if err != nil {
		return "", err
	}
	return "vcd.organization/" + id, nil
}

func (v *mqlVcdOrganization) GetSettings() (interface{}, error) {
	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	name, err := v.Name()
	if err != nil {
		return nil, err
	}

	adminOrgClient, err := client.GetAdminOrgByName(name)
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(adminOrgClient.AdminOrg.OrgSettings)
}

// https://developer.vmware.com/apis/1260/vmware-cloud-director/doc/doc/types/OrgLdapSettingsType.html
func (v *mqlVcdOrganization) GetLdapConfiguration() (interface{}, error) {
	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	name, err := v.Name()
	if err != nil {
		return nil, err
	}

	adminOrgClient, err := client.GetAdminOrgByName(name)
	if err != nil {
		return nil, err
	}

	ldapConfig, err := adminOrgClient.GetLdapConfiguration()
	if err != nil {
		return nil, err
	}

	return newMqlVcdOrgLdapConfiguration(v.MotorRuntime, ldapConfig)
}

func newMqlVcdOrgLdapConfiguration(runtime *resources.Runtime, org *types.OrgLdapSettingsType) (interface{}, error) {
	hostname := ""
	username := ""
	realm := ""
	if org.CustomOrgLdapSettings != nil {
		hostname = org.CustomOrgLdapSettings.HostName
		username = org.CustomOrgLdapSettings.Username
		realm = org.CustomOrgLdapSettings.Realm
	}
	return runtime.CreateResource("vcd.organization.ldapSettings",
		"id", org.HREF,
		"customUsersOu", org.CustomUsersOu,
		"orgLdapMode", org.OrgLdapMode,
		"hostname", hostname,
		"username", username,
		"realm", realm,
	)
}

func (v *mqlVcdOrganizationLdapSettings) id() (string, error) {
	id, err := v.Id()
	if err != nil {
		return "", err
	}
	return "vcd.organization.ldapSettings/" + id, nil
}

// https://developer.vmware.com/apis/1260/vmware-cloud-director/doc/doc/types/QueryResultAdminVMRecordType.html
func (v *mqlVcdOrganization) GetVms() ([]interface{}, error) {
	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	name, err := v.Name()
	if err != nil {
		return nil, err
	}

	orgClient, err := client.GetOrgByName(name)
	if err != nil {
		return nil, err
	}
	vmList, err := orgClient.QueryVmList(types.VmQueryFilterAll)
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range vmList {
		entry, err := newMqlVcdVm(v.MotorRuntime, vmList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}
	return list, nil
}

func newMqlVcdVm(runtime *resources.Runtime, vm *types.QueryResultVMRecordType) (interface{}, error) {
	totalStorage := int64(0)
	if vm.TotalStorageAllocatedMb != "" {
		// this value can be ""
		num, err := strconv.Atoi(vm.TotalStorageAllocatedMb)
		if err == nil {
			totalStorage = int64(num)
		} else {
			log.Error().Err(err).Msg("value: " + vm.TotalStorageAllocatedMb)
		}
	}

	return runtime.CreateResource("vcd.vm",
		"id", vm.ID,
		"name", vm.Name,
		"containerName", vm.ContainerName,
		"containerID", vm.ContainerID,
		"ownerId", vm.Owner,
		"ownerName", vm.OwnerName,
		"isDeleted", vm.Deleted,
		"guestOs", vm.GuestOS,
		"numberOfCpus", int64(vm.Cpus),
		"memoryMB", int64(vm.MemoryMB),
		"status", vm.Status,
		"networkName", vm.NetworkName,
		"ipAddress", vm.IpAddress,
		"isBusy", vm.Busy,
		"isDeployed", vm.Deployed,
		"isPublished", vm.Published,
		"catalogName", vm.CatalogName,
		"hardwareVersion", int64(vm.HardwareVersion),
		"vmToolsStatus", vm.VmToolsStatus,
		"isInMaintenanceMode", vm.MaintenanceMode,
		"isAutoNature", vm.AutoNature,
		"storageProfileName", vm.StorageProfileName,
		"gcStatus", vm.GcStatus,
		"isComputePolicyCompliant", vm.IsComputePolicyCompliant,
		"encrypted", vm.Encrypted,
		"totalStorageAllocatedMb", totalStorage,
		"isExpired", vm.IsExpired,
		"hostName", vm.HostName,
	)
}

func (v *mqlVcdVm) id() (string, error) {
	id, err := v.Name()
	if err != nil {
		return "", err
	}
	return "vcd.vm/" + id, nil
}

// https://developer.vmware.com/apis/72/vmware-cloud-director/doc/doc/types/OrganizationRightsType.html
func (v *mqlVcdOrganization) GetRights() ([]interface{}, error) {
	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	name, err := v.Name()
	if err != nil {
		return nil, err
	}

	adminOrgClient, err := client.GetAdminOrgByName(name)
	if err != nil {
		return nil, err
	}

	rightList, err := adminOrgClient.GetAllRights(nil)
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range rightList {
		entry, err := newMqlVcdRight(v.MotorRuntime, rightList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}
	return list, nil
}

func newMqlVcdRight(runtime *resources.Runtime, right *types.Right) (interface{}, error) {
	return runtime.CreateResource("vcd.right",
		"id", right.ID,
		"name", right.Name,
		"description", right.Description,
		"bundleKey", right.BundleKey,
		"category", right.Category,
		"serviceNamespace", right.ServiceNamespace,
		"rightType", right.RightType,
	)
}

func (v *mqlVcdRight) id() (string, error) {
	id, err := v.Name()
	if err != nil {
		return "", err
	}
	return "vcd.right/" + id, nil
}

// see https://developer.vmware.com/apis/1260/vmware-cloud-director/doc/doc//types/AdminVdcType.html
func (v *mqlVcdOrganization) GetVdcs() ([]interface{}, error) {
	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	name, err := v.Name()
	if err != nil {
		return nil, err
	}

	adminOrgClient, err := client.GetAdminOrgByName(name)
	if err != nil {
		return nil, err
	}

	vdcList, err := adminOrgClient.GetAllVDCs(false)
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range vdcList {
		entry, err := newMqlVcdVdc(v.MotorRuntime, vdcList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}
	return list, nil
}

func newMqlVcdVdc(runtime *resources.Runtime, dataCenter *govcd.Vdc) (interface{}, error) {
	res, err := runtime.CreateResource("vcd.vdc",
		"id", dataCenter.Vdc.ID,
		"name", dataCenter.Vdc.Name,
		"description", dataCenter.Vdc.Description,
		"status", int64(dataCenter.Vdc.Status),
		"allocationModel", dataCenter.Vdc.AllocationModel,
		"nicQuota", int64(dataCenter.Vdc.NicQuota),
		"networkQuota", int64(dataCenter.Vdc.NetworkQuota),
		"usedNetworkCount", int64(dataCenter.Vdc.UsedNetworkCount),
		"vmQuota", int64(dataCenter.Vdc.VMQuota),
		"isEnabled", dataCenter.Vdc.IsEnabled,
	)
	if err != nil {
		return nil, err
	}

	res.MqlResource().Cache.Store("_vdc", &resources.CacheEntry{Data: dataCenter})
	return res, nil
}

func (v *mqlVcdVdc) id() (string, error) {
	id, err := v.Name()
	if err != nil {
		return "", err
	}
	return "vcd.vdc/" + id, nil
}

func (v *mqlVcdOrganization) GetVdcGroups() ([]interface{}, error) {
	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	name, err := v.Name()
	if err != nil {
		return nil, err
	}

	adminOrgClient, err := client.GetAdminOrgByName(name)
	if err != nil {
		return nil, err
	}

	vdcGroupList, err := adminOrgClient.GetAllVdcGroups(nil)
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range vdcGroupList {
		entry, err := newMqlVcdVdcGroup(v.MotorRuntime, vdcGroupList[i].VdcGroup)
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}
	return list, nil
}

func newMqlVcdVdcGroup(runtime *resources.Runtime, grp *types.VdcGroup) (interface{}, error) {
	return runtime.CreateResource("vcd.vdcGroup",
		"name", grp.Name,
		"description", grp.Description,
		"localEgress", grp.LocalEgress,
		"status", grp.Status,
		"type", grp.Type,
		"universalNetworkingEnabled", grp.UniversalNetworkingEnabled,
		"dfwEnabled", grp.DfwEnabled,
	)
}

func (v *mqlVcdVdcGroup) id() (string, error) {
	id, err := v.Name()
	if err != nil {
		return "", err
	}
	return "vcd.vdcGroup/" + id, nil
}

// https://developer.vmware.com/apis/1260/vmware-cloud-director/doc/doc//types/RoleType.html
func (v *mqlVcdOrganization) GetRoles() ([]interface{}, error) {
	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	name, err := v.Name()
	if err != nil {
		return nil, err
	}

	adminOrgClient, err := client.GetAdminOrgByName(name)
	if err != nil {
		return nil, err
	}

	rolesList, err := adminOrgClient.GetAllRoles(nil)
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range rolesList {
		entry, err := newMqlVcdRole(v.MotorRuntime, rolesList[i].Role)
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}
	return list, nil
}

func newMqlVcdRole(runtime *resources.Runtime, role *types.Role) (interface{}, error) {
	return runtime.CreateResource("vcd.role",
		"id", role.ID,
		"name", role.Name,
		"description", role.Description,
	)
}

func (v *mqlVcdRole) id() (string, error) {
	id, err := v.Id()
	if err != nil {
		return "", err
	}
	return "vcd.role/" + id, nil
}
