// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	compute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
	"golang.org/x/sync/errgroup"
)

func (a *mqlAzureSubscriptionComputeService) id() (string, error) {
	return "azure.subscription.compute/" + a.SubscriptionId.Data, nil
}

func getState(vm compute.VirtualMachineInstanceView) string {
	if vm.Statuses == nil {
		return "unknown"
	}
	state := "unknown"
	for _, s := range vm.Statuses {
		if s.Code != nil && *s.Code == "PowerState/running" {
			state = "running"
		}
		if s.Code != nil && *s.Code == "PowerState/deallocated" {
			state = "stopped"
		}
	}
	return state
}

func initAzureSubscriptionComputeService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionComputeService) vms() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)

	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	// list compute instances
	vmClient, err := compute.NewVirtualMachinesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := vmClient.NewListAllPager(&compute.VirtualMachinesClientListAllOptions{})
	res := []any{}
	for pager.More() {
		vms, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vm := range vms.Value {
			properties, err := convert.JsonToDict(vm.Properties)
			if err != nil {
				return nil, err
			}

			var encryptionAtHost, secureBootEnabled, vtpmEnabled, proxyAgentEnabled *bool
			var securityType *string
			if vm.Properties != nil && vm.Properties.SecurityProfile != nil {
				sp := vm.Properties.SecurityProfile
				encryptionAtHost = sp.EncryptionAtHost
				securityType = (*string)(sp.SecurityType)
				if sp.UefiSettings != nil {
					secureBootEnabled = sp.UefiSettings.SecureBootEnabled
					vtpmEnabled = sp.UefiSettings.VTpmEnabled
				}
				if sp.ProxyAgentSettings != nil {
					proxyAgentEnabled = sp.ProxyAgentSettings.Enabled
				}
			}

			var computerName, adminUsername, licenseType *string
			var vmId, provisioningState *string
			var timeCreated *time.Time
			var disablePasswordAuth, provisionVMAgent, enableAutomaticUpdates *bool
			var patchMode string
			var sshPublicKeys []any
			if vm.Properties != nil {
				licenseType = vm.Properties.LicenseType
				vmId = vm.Properties.VMID
				provisioningState = vm.Properties.ProvisioningState
				timeCreated = vm.Properties.TimeCreated
				if osp := vm.Properties.OSProfile; osp != nil {
					computerName = osp.ComputerName
					adminUsername = osp.AdminUsername
					if lc := osp.LinuxConfiguration; lc != nil {
						disablePasswordAuth = lc.DisablePasswordAuthentication
						provisionVMAgent = lc.ProvisionVMAgent
						if lc.SSH != nil {
							for _, k := range lc.SSH.PublicKeys {
								if k == nil {
									continue
								}
								entry := map[string]any{}
								if k.Path != nil {
									entry["path"] = *k.Path
								}
								if k.KeyData != nil {
									entry["keyData"] = *k.KeyData
								}
								sshPublicKeys = append(sshPublicKeys, entry)
							}
						}
						if lc.PatchSettings != nil && lc.PatchSettings.PatchMode != nil {
							patchMode = string(*lc.PatchSettings.PatchMode)
						}
					}
					if wc := osp.WindowsConfiguration; wc != nil {
						provisionVMAgent = wc.ProvisionVMAgent
						enableAutomaticUpdates = wc.EnableAutomaticUpdates
						if wc.PatchSettings != nil && wc.PatchSettings.PatchMode != nil {
							patchMode = string(*wc.PatchSettings.PatchMode)
						}
					}
				}
			}

			var id *string
			if vm.ID != nil {
				normalized := strings.ToLower(*vm.ID)
				id = &normalized
			}
			mqlAzureVm, err := CreateResource(a.MqlRuntime, "azure.subscription.computeService.vm",
				map[string]*llx.RawData{
					"id":                            llx.StringDataPtr(id),
					"name":                          llx.StringDataPtr(vm.Name),
					"location":                      llx.StringDataPtr(vm.Location),
					"zones":                         llx.ArrayData(convert.SliceStrPtrToInterface(vm.Zones), types.String),
					"tags":                          llx.MapData(convert.PtrMapStrToInterface(vm.Tags), types.String),
					"type":                          llx.StringDataPtr(vm.Type),
					"properties":                    llx.DictData(properties),
					"encryptionAtHost":              llx.BoolDataPtr(encryptionAtHost),
					"securityType":                  llx.StringDataPtr(securityType),
					"secureBootEnabled":             llx.BoolDataPtr(secureBootEnabled),
					"vtpmEnabled":                   llx.BoolDataPtr(vtpmEnabled),
					"proxyAgentEnabled":             llx.BoolDataPtr(proxyAgentEnabled),
					"computerName":                  llx.StringDataPtr(computerName),
					"adminUsername":                 llx.StringDataPtr(adminUsername),
					"licenseType":                   llx.StringDataPtr(licenseType),
					"managedBy":                     llx.StringDataPtr(vm.ManagedBy),
					"vmId":                          llx.StringDataPtr(vmId),
					"provisioningState":             llx.StringDataPtr(provisioningState),
					"timeCreated":                   llx.TimeDataPtr(timeCreated),
					"sshPublicKeys":                 llx.ArrayData(sshPublicKeys, types.Dict),
					"disablePasswordAuthentication": llx.BoolDataPtr(disablePasswordAuth),
					"provisionVMAgent":              llx.BoolDataPtr(provisionVMAgent),
					"enableAutomaticUpdates":        llx.BoolDataPtr(enableAutomaticUpdates),
					"patchMode":                     llx.StringData(patchMode),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureVm)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionComputeServiceVm) state() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	// id is a Azure resource ID
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return "", err
	}

	vm, err := resourceID.Component("virtualMachines")
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	token := conn.Token()
	if err != nil {
		return "", err
	}

	client, err := compute.NewVirtualMachinesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return "", err
	}

	view, err := client.InstanceView(ctx, resourceID.ResourceGroup, vm, &compute.VirtualMachinesClientInstanceViewOptions{})
	if err != nil {
		return "", err
	}
	return getState(view.VirtualMachineInstanceView), nil
}

func (a *mqlAzureSubscriptionComputeServiceVm) isRunning() (bool, error) {
	state := a.GetState()
	if state.Error != nil {
		return false, state.Error
	}
	return state.Data == "running", nil
}

func (a *mqlAzureSubscriptionComputeServiceVm) extensions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	// id is a Azure resource ID
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	vm, err := resourceID.Component("virtualMachines")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token := conn.Token()
	if err != nil {
		return nil, err
	}

	client, err := compute.NewVirtualMachineExtensionsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	extensions, err := client.List(ctx, resourceID.ResourceGroup, vm, &compute.VirtualMachineExtensionsClientListOptions{})
	if err != nil {
		return nil, err
	}

	res := []any{}

	if extensions.Value == nil {
		return res, nil
	}

	list := extensions.Value

	for i := range list {
		entry := list[i]

		dict, err := convert.JsonToDict(entry.Properties)
		if err != nil {
			return nil, err
		}

		res = append(res, dict)
	}

	return res, nil
}

func (a *mqlAzureSubscriptionComputeService) disks() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)

	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := compute.NewDisksClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(&compute.DisksClientListOptions{})
	if err != nil {
		return nil, err
	}

	res := []any{}
	for pager.More() {
		disks, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, disk := range disks.Value {
			mqlAzureDisk, err := diskToMql(a.MqlRuntime, *disk)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDisk)
		}
	}

	return res, nil
}

func diskToMql(runtime *plugin.Runtime, disk compute.Disk) (*mqlAzureSubscriptionComputeServiceDisk, error) {
	properties, err := convert.JsonToDict(disk.Properties)
	if err != nil {
		return nil, err
	}

	sku, err := convert.JsonToDict(disk.SKU)
	if err != nil {
		return nil, err
	}

	managedByExtended := []any{}
	for _, mbe := range disk.ManagedByExtended {
		if mbe != nil {
			managedByExtended = append(managedByExtended, *mbe)
		}
	}
	zones := []any{}
	for _, z := range disk.Zones {
		if z != nil {
			zones = append(zones, *z)
		}
	}

	args := map[string]*llx.RawData{
		"id":                llx.StringDataPtr(disk.ID),
		"name":              llx.StringDataPtr(disk.Name),
		"location":          llx.StringDataPtr(disk.Location),
		"tags":              llx.MapData(convert.PtrMapStrToInterface(disk.Tags), types.String),
		"type":              llx.StringDataPtr(disk.Type),
		"managedBy":         llx.StringDataPtr(disk.ManagedBy),
		"managedByExtended": llx.ArrayData(managedByExtended, types.String),
		"zones":             llx.ArrayData(zones, types.String),
		"sku":               llx.DictData(sku),
		"properties":        llx.DictData(properties),
	}

	if disk.Properties != nil {
		if disk.Properties.NetworkAccessPolicy != nil {
			args["networkAccessPolicy"] = llx.StringData(string(*disk.Properties.NetworkAccessPolicy))
		}
		if disk.Properties.PublicNetworkAccess != nil {
			args["publicNetworkAccess"] = llx.StringData(string(*disk.Properties.PublicNetworkAccess))
		}
		if disk.Properties.Encryption != nil {
			if disk.Properties.Encryption.Type != nil {
				args["encryptionType"] = llx.StringData(string(*disk.Properties.Encryption.Type))
			}
			args["diskEncryptionSetId"] = llx.StringDataPtr(disk.Properties.Encryption.DiskEncryptionSetID)
		}
		if disk.Properties.DataAccessAuthMode != nil {
			args["dataAccessAuthMode"] = llx.StringData(string(*disk.Properties.DataAccessAuthMode))
		}
		if disk.Properties.DiskState != nil {
			args["diskState"] = llx.StringData(string(*disk.Properties.DiskState))
		}
		args["provisioningState"] = llx.StringDataPtr(disk.Properties.ProvisioningState)
		args["timeCreated"] = llx.TimeDataPtr(disk.Properties.TimeCreated)
		args["uniqueId"] = llx.StringDataPtr(disk.Properties.UniqueID)
		args["burstingEnabled"] = llx.BoolDataPtr(disk.Properties.BurstingEnabled)
		args["diskSizeBytes"] = llx.IntDataPtr(disk.Properties.DiskSizeBytes)
		args["diskIopsReadWrite"] = llx.IntDataPtr(disk.Properties.DiskIOPSReadWrite)
		args["diskMbpsReadWrite"] = llx.IntDataPtr(disk.Properties.DiskMBpsReadWrite)
		args["diskIopsReadOnly"] = llx.IntDataPtr(disk.Properties.DiskIOPSReadOnly)
		args["diskMbpsReadOnly"] = llx.IntDataPtr(disk.Properties.DiskMBpsReadOnly)
		args["maxShares"] = llx.IntDataPtr(disk.Properties.MaxShares)
		if disk.Properties.HyperVGeneration != nil {
			args["hyperVGeneration"] = llx.StringData(string(*disk.Properties.HyperVGeneration))
		}
		args["tier"] = llx.StringDataPtr(disk.Properties.Tier)
		args["supportsHibernation"] = llx.BoolDataPtr(disk.Properties.SupportsHibernation)
		args["diskAccessId"] = llx.StringDataPtr(disk.Properties.DiskAccessID)
	}

	res, err := CreateResource(runtime, "azure.subscription.computeService.disk", args)
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionComputeServiceDisk), nil
}

func (a *mqlAzureSubscriptionComputeServiceVm) osDisk() (*mqlAzureSubscriptionComputeServiceDisk, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	propertiesDict := a.Properties.Data
	data, err := json.Marshal(propertiesDict)
	if err != nil {
		return nil, err
	}

	var properties compute.VirtualMachineProperties
	err = json.Unmarshal(data, &properties)
	if err != nil {
		return nil, err
	}

	if properties.StorageProfile == nil || properties.StorageProfile.OSDisk == nil || properties.StorageProfile.OSDisk.ManagedDisk == nil || properties.StorageProfile.OSDisk.ManagedDisk.ID == nil {
		return nil, errors.New("could not determine os disk from vm storage profile")
	}

	resourceID, err := ParseResourceID(*properties.StorageProfile.OSDisk.ManagedDisk.ID)
	if err != nil {
		return nil, err
	}

	diskName, err := resourceID.Component("disks")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token := conn.Token()

	client, err := compute.NewDisksClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	disk, err := client.Get(ctx, resourceID.ResourceGroup, diskName, &compute.DisksClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return diskToMql(a.MqlRuntime, disk.Disk)
}

func (a *mqlAzureSubscriptionComputeServiceVm) dataDisks() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	propertiesDict := a.Properties.Data
	data, err := json.Marshal(propertiesDict)
	if err != nil {
		return nil, err
	}

	token := conn.Token()

	var properties compute.VirtualMachineProperties
	err = json.Unmarshal(data, &properties)
	if err != nil {
		return nil, err
	}

	if properties.StorageProfile == nil || properties.StorageProfile.DataDisks == nil {
		return nil, errors.New("could not determine data disks from vm storage profile")
	}

	dataDisks := properties.StorageProfile.DataDisks

	// Pre-validate all disks (parse IDs, build clients) before spawning any
	// goroutines so that an early error can't leak in-flight workers.
	type diskFetch struct {
		client *compute.DisksClient
		rg     string
		name   string
		index  int
	}
	clientsBySub := map[string]*compute.DisksClient{}
	disks := make([]compute.Disk, len(dataDisks))
	fetches := make([]diskFetch, 0, len(dataDisks))
	for i, dd := range dataDisks {
		if dd.ManagedDisk == nil || dd.ManagedDisk.ID == nil {
			continue
		}
		resourceID, err := ParseResourceID(*dd.ManagedDisk.ID)
		if err != nil {
			return nil, err
		}
		diskName, err := resourceID.Component("disks")
		if err != nil {
			return nil, err
		}
		client, ok := clientsBySub[resourceID.SubscriptionID]
		if !ok {
			client, err = compute.NewDisksClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
				ClientOptions: conn.ClientOptions(),
			})
			if err != nil {
				return nil, err
			}
			clientsBySub[resourceID.SubscriptionID] = client
		}
		fetches = append(fetches, diskFetch{client: client, rg: resourceID.ResourceGroup, name: diskName, index: i})
	}

	// Azure has no batch get-by-id endpoint for disks, so concurrent per-disk
	// Gets in a bounded errgroup is the cheapest fix.
	ctx := context.Background()
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(10)
	for _, f := range fetches {
		g.Go(func() error {
			resp, err := f.client.Get(gctx, f.rg, f.name, &compute.DisksClientGetOptions{})
			if err != nil {
				return err
			}
			disks[f.index] = resp.Disk
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	res := []any{}
	for _, disk := range disks {
		if disk.ID == nil {
			continue
		}
		mqlDisk, err := diskToMql(a.MqlRuntime, disk)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlDisk)
	}

	return res, nil
}

func (a *mqlAzureSubscriptionComputeServiceVm) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionComputeServiceDisk) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionComputeServiceVm) publicIpAddresses() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	token := conn.Token()
	resourceId, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	subId := resourceId.SubscriptionID
	props := a.GetProperties()
	if props.Error != nil {
		return nil, props.Error
	}

	propsDict := (props.Data).(map[string]any)
	networkInterface, ok := propsDict["networkProfile"]
	if !ok {
		return nil, errors.New("cannot find network profile on vm, not retrieving ip addresses")
	}
	var networkInterfaces compute.NetworkProfile

	data, err := json.Marshal(networkInterface)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), &networkInterfaces)
	if err != nil {
		return nil, err
	}
	res := []any{}

	ctx := context.Background()
	nicClient, err := network.NewInterfacesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	ipClient, err := network.NewPublicIPAddressesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	// Phase 1: pre-validate NIC IDs, then fetch all NICs concurrently. Per-NIC
	// Get is the only API option here (no batch endpoint), so a bounded
	// errgroup is the cheapest fix that preserves semantics.
	type nicFetch struct {
		rg    string
		name  string
		index int
	}
	nicFetches := make([]nicFetch, 0, len(networkInterfaces.NetworkInterfaces))
	for i, iface := range networkInterfaces.NetworkInterfaces {
		if iface.ID == nil {
			continue
		}
		resource, err := ParseResourceID(*iface.ID)
		if err != nil {
			return nil, err
		}
		name, err := resource.Component("networkInterfaces")
		if err != nil {
			return nil, err
		}
		nicFetches = append(nicFetches, nicFetch{rg: resource.ResourceGroup, name: name, index: i})
	}

	nics := make([]network.Interface, len(networkInterfaces.NetworkInterfaces))
	g1, gctx1 := errgroup.WithContext(ctx)
	g1.SetLimit(10)
	for _, f := range nicFetches {
		g1.Go(func() error {
			resp, err := nicClient.Get(gctx1, f.rg, f.name, &network.InterfacesClientGetOptions{})
			if err != nil {
				return err
			}
			nics[f.index] = resp.Interface
			return nil
		})
	}
	if err := g1.Wait(); err != nil {
		return nil, err
	}

	// Phase 2: collect all public IP IDs across NICs, then fetch concurrently.
	type ipFetch struct {
		rg   string
		name string
	}
	var fetches []ipFetch
	for _, ni := range nics {
		if ni.Properties == nil {
			continue
		}
		for _, config := range ni.Properties.IPConfigurations {
			if config.Properties == nil || config.Properties.PublicIPAddress == nil || config.Properties.PublicIPAddress.ID == nil {
				continue
			}
			publicIPID := *config.Properties.PublicIPAddress.ID
			publicIpResource, err := ParseResourceID(publicIPID)
			if err != nil {
				return nil, errors.New("invalid network information for resource " + publicIPID)
			}
			ipAddrName, err := publicIpResource.Component("publicIPAddresses")
			if err != nil {
				return nil, errors.New("invalid network information for resource " + publicIPID)
			}
			fetches = append(fetches, ipFetch{rg: publicIpResource.ResourceGroup, name: ipAddrName})
		}
	}

	ipResults := make([]network.PublicIPAddress, len(fetches))
	g2, gctx2 := errgroup.WithContext(ctx)
	g2.SetLimit(10)
	for i, f := range fetches {
		g2.Go(func() error {
			resp, err := ipClient.Get(gctx2, f.rg, f.name, &network.PublicIPAddressesClientGetOptions{})
			if err != nil {
				return err
			}
			ipResults[i] = resp.PublicIPAddress
			return nil
		})
	}
	if err := g2.Wait(); err != nil {
		return nil, err
	}

	for _, ipAddress := range ipResults {
		mqlIpAddress, err := azureIpToMql(a.MqlRuntime, ipAddress)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlIpAddress)
	}

	return res, nil
}

func initAzureSubscriptionComputeServiceVm(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure compute vm instance")
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.computeService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	computeSvc := res.(*mqlAzureSubscriptionComputeService)
	vms := computeSvc.GetVms()
	if vms.Error != nil {
		return nil, nil, vms.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range vms.Data {
		vm := entry.(*mqlAzureSubscriptionComputeServiceVm)
		if vm.Id.Data == id {
			return args, vm, nil
		}
	}

	return nil, nil, errors.New("azure compute instance does not exist")
}

func (a *mqlAzureSubscriptionComputeServiceDiskEncryptionSet) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionComputeServiceDiskAccess) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionComputeService) diskEncryptionSets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := compute.NewDiskEncryptionSetsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list disk encryption sets due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, des := range page.Value {
			if des == nil {
				continue
			}

			identity, err := convert.JsonToDict(des.Identity)
			if err != nil {
				return nil, err
			}

			var encryptionType, provisioningState string
			var activeKeyUrl, activeKeySourceVaultId string
			var rotationToLatestKeyVersionEnabled *bool
			var lastKeyRotationTimestamp *time.Time

			if des.Properties != nil {
				props := des.Properties
				if props.EncryptionType != nil {
					encryptionType = string(*props.EncryptionType)
				}
				if props.ActiveKey != nil {
					if props.ActiveKey.KeyURL != nil {
						activeKeyUrl = *props.ActiveKey.KeyURL
					}
					if props.ActiveKey.SourceVault != nil && props.ActiveKey.SourceVault.ID != nil {
						activeKeySourceVaultId = *props.ActiveKey.SourceVault.ID
					}
				}
				rotationToLatestKeyVersionEnabled = props.RotationToLatestKeyVersionEnabled
				lastKeyRotationTimestamp = props.LastKeyRotationTimestamp
				if props.ProvisioningState != nil {
					provisioningState = *props.ProvisioningState
				}
			}

			mqlDes, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionComputeServiceDiskEncryptionSet,
				map[string]*llx.RawData{
					"id":                                llx.StringDataPtr(des.ID),
					"name":                              llx.StringDataPtr(des.Name),
					"location":                          llx.StringDataPtr(des.Location),
					"tags":                              llx.MapData(convert.PtrMapStrToInterface(des.Tags), types.String),
					"type":                              llx.StringDataPtr(des.Type),
					"identity":                          llx.DictData(identity),
					"encryptionType":                    llx.StringData(encryptionType),
					"activeKeyUrl":                      llx.StringData(activeKeyUrl),
					"activeKeySourceVaultId":            llx.StringData(activeKeySourceVaultId),
					"rotationToLatestKeyVersionEnabled": llx.BoolDataPtr(rotationToLatestKeyVersionEnabled),
					"lastKeyRotationTimestamp":          llx.TimeDataPtr(lastKeyRotationTimestamp),
					"provisioningState":                 llx.StringData(provisioningState),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlDes)
		}
	}

	return res, nil
}

type mqlAzureSubscriptionComputeServiceDiskAccessInternal struct {
	cachePrivateEndpointConnections []*compute.PrivateEndpointConnection
}

func (a *mqlAzureSubscriptionComputeService) diskAccesses() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := compute.NewDiskAccessesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list disk accesses due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, da := range page.Value {
			if da == nil {
				continue
			}

			var provisioningState string
			var timeCreated *time.Time
			var privateEndpointConns []*compute.PrivateEndpointConnection

			if da.Properties != nil {
				if da.Properties.ProvisioningState != nil {
					provisioningState = *da.Properties.ProvisioningState
				}
				timeCreated = da.Properties.TimeCreated
				privateEndpointConns = da.Properties.PrivateEndpointConnections
			}

			mqlDa, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionComputeServiceDiskAccess,
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(da.ID),
					"name":              llx.StringDataPtr(da.Name),
					"location":          llx.StringDataPtr(da.Location),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(da.Tags), types.String),
					"type":              llx.StringDataPtr(da.Type),
					"provisioningState": llx.StringData(provisioningState),
					"timeCreated":       llx.TimeDataPtr(timeCreated),
				})
			if err != nil {
				return nil, err
			}

			// Cache private endpoint connections for lazy loading
			mqlDaTyped := mqlDa.(*mqlAzureSubscriptionComputeServiceDiskAccess)
			mqlDaTyped.cachePrivateEndpointConnections = privateEndpointConns

			res = append(res, mqlDa)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionComputeServiceDiskAccess) privateEndpointConnections() ([]any, error) {
	var res []any
	if a.cachePrivateEndpointConnections == nil {
		return res, nil
	}

	for _, entry := range a.cachePrivateEndpointConnections {
		if entry == nil {
			continue
		}

		// Extract name and type from ID
		var name, resType string
		if entry.ID != nil {
			connResourceID, err := ParseResourceID(*entry.ID)
			if err == nil {
				if nameComp, err := connResourceID.Component("privateEndpointConnections"); err == nil {
					name = nameComp
				}
				if connResourceID.Provider != "" {
					resType = connResourceID.Provider + "/diskAccesses/privateEndpointConnections"
				}
			}
			if name == "" {
				parts := strings.Split(*entry.ID, "/")
				if len(parts) > 0 {
					name = parts[len(parts)-1]
				}
			}
		}
		if resType == "" {
			resType = "Microsoft.Compute/diskAccesses/privateEndpointConnections"
		}

		privateEndpoint := map[string]*llx.RawData{
			"__id": llx.StringDataPtr(entry.ID),
			"id":   llx.StringDataPtr(entry.ID),
		}
		if name != "" {
			privateEndpoint["name"] = llx.StringData(name)
		}
		privateEndpoint["type"] = llx.StringData(resType)

		if entry.Properties != nil {
			props := entry.Properties
			propsMap, err := convert.JsonToDict(props)
			if err != nil {
				return nil, err
			}

			privateEndpoint["properties"] = llx.DictData(propsMap)

			if props.PrivateEndpoint != nil {
				privateEndpoint["privateEndpointId"] = llx.StringDataPtr(props.PrivateEndpoint.ID)
			}
			if props.PrivateLinkServiceConnectionState != nil {
				stateArgs := map[string]*llx.RawData{}
				if props.PrivateLinkServiceConnectionState.ActionsRequired != nil {
					stateArgs["actionsRequired"] = llx.StringDataPtr(props.PrivateLinkServiceConnectionState.ActionsRequired)
				}
				if props.PrivateLinkServiceConnectionState.Description != nil {
					stateArgs["description"] = llx.StringDataPtr(props.PrivateLinkServiceConnectionState.Description)
				}
				if props.PrivateLinkServiceConnectionState.Status != nil {
					stateArgs["status"] = llx.StringData(string(*props.PrivateLinkServiceConnectionState.Status))
				}
				stateRes, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionPrivateEndpointConnectionConnectionState, stateArgs)
				if err != nil {
					return nil, err
				}
				privateEndpoint["privateLinkServiceConnectionState"] = llx.ResourceData(stateRes, ResourceAzureSubscriptionPrivateEndpointConnectionConnectionState)
			}
			if props.ProvisioningState != nil {
				privateEndpoint["provisioningState"] = llx.StringData(string(*props.ProvisioningState))
			}
		}

		mqlRes, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionPrivateEndpointConnection, privateEndpoint)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlRes)
	}

	return res, nil
}
