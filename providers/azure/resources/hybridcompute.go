// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	hybridcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/hybridcompute/armhybridcompute/v2"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionComputeService) hybridMachines() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)

	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := hybridcompute.NewMachinesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, m := range page.Value {
			if m == nil {
				continue
			}

			properties, err := convert.JsonToDict(m.Properties)
			if err != nil {
				return nil, err
			}
			systemDataDict, err := convert.JsonToDict(m.SystemData)
			if err != nil {
				return nil, err
			}

			var (
				status                                                                               *string
				lastStatusChange                                                                     *time.Time
				agentVersion, osName, osSku, osEdition, osVersion, osType                            *string
				displayName, machineFqdn, dnsFqdn, domainName, adFqdn                                *string
				vmID, vmUUID, provisioningState, parentClusterResourceID, privateLinkScopeResourceID *string
				cloudMetadataDict, licenseProfileDict                                                map[string]any
				detectedProperties                                                                   map[string]any
			)

			if p := m.Properties; p != nil {
				lastStatusChange = p.LastStatusChange
				status = stringEnumPtr(p.Status)
				agentVersion = p.AgentVersion
				osName = p.OSName
				osSku = p.OSSKU
				osEdition = p.OSEdition
				osVersion = p.OSVersion
				osType = p.OSType
				displayName = p.DisplayName
				machineFqdn = p.MachineFqdn
				dnsFqdn = p.DNSFqdn
				domainName = p.DomainName
				adFqdn = p.AdFqdn
				vmID = p.VMID
				vmUUID = p.VMUUID
				provisioningState = p.ProvisioningState
				parentClusterResourceID = p.ParentClusterResourceID
				privateLinkScopeResourceID = p.PrivateLinkScopeResourceID

				cloudMetadataDict, err = convert.JsonToDict(p.CloudMetadata)
				if err != nil {
					return nil, err
				}
				licenseProfileDict, err = convert.JsonToDict(p.LicenseProfile)
				if err != nil {
					return nil, err
				}

				if p.DetectedProperties != nil {
					detectedProperties = make(map[string]any, len(p.DetectedProperties))
					for k, v := range p.DetectedProperties {
						if v != nil {
							detectedProperties[k] = *v
						} else {
							detectedProperties[k] = ""
						}
					}
				}
			}

			var id *string
			if m.ID != nil {
				normalized := strings.ToLower(*m.ID)
				id = &normalized
			}

			mqlMachine, err := CreateResource(a.MqlRuntime, "azure.subscription.computeService.hybridMachine",
				map[string]*llx.RawData{
					"id":                         llx.StringDataPtr(id),
					"name":                       llx.StringDataPtr(m.Name),
					"location":                   llx.StringDataPtr(m.Location),
					"tags":                       llx.MapData(convert.PtrMapStrToInterface(m.Tags), types.String),
					"type":                       llx.StringDataPtr(m.Type),
					"status":                     llx.StringDataPtr(status),
					"lastStatusChange":           llx.TimeDataPtr(lastStatusChange),
					"agentVersion":               llx.StringDataPtr(agentVersion),
					"osName":                     llx.StringDataPtr(osName),
					"osSku":                      llx.StringDataPtr(osSku),
					"osEdition":                  llx.StringDataPtr(osEdition),
					"osVersion":                  llx.StringDataPtr(osVersion),
					"osType":                     llx.StringDataPtr(osType),
					"displayName":                llx.StringDataPtr(displayName),
					"machineFqdn":                llx.StringDataPtr(machineFqdn),
					"dnsFqdn":                    llx.StringDataPtr(dnsFqdn),
					"domainName":                 llx.StringDataPtr(domainName),
					"adFqdn":                     llx.StringDataPtr(adFqdn),
					"vmId":                       llx.StringDataPtr(vmID),
					"vmUuid":                     llx.StringDataPtr(vmUUID),
					"provisioningState":          llx.StringDataPtr(provisioningState),
					"parentClusterResourceId":    llx.StringDataPtr(parentClusterResourceID),
					"privateLinkScopeResourceId": llx.StringDataPtr(privateLinkScopeResourceID),
					"detectedProperties":         llx.MapData(detectedProperties, types.String),
					"cloudMetadata":              llx.DictData(cloudMetadataDict),
					"licenseProfile":             llx.DictData(licenseProfileDict),
					"properties":                 llx.DictData(properties),
					"systemData":                 llx.DictData(systemDataDict),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlMachine)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionComputeServiceHybridMachine) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionComputeServiceHybridMachineExtension) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionComputeServiceHybridMachine) extensions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	// id is pre-normalized to lowercase by the lister (see hybridMachines), so
	// downstream ARM lookups via Component("machines") receive a lowercase name.
	// ARM is case-insensitive on resource names, so this is safe.
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	machineName, err := resourceID.Component("machines")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token := conn.Token()

	client, err := hybridcompute.NewMachineExtensionsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(resourceID.ResourceGroup, machineName, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ext := range page.Value {
			if ext == nil {
				continue
			}
			mqlExt, err := hybridMachineExtensionToMql(a.MqlRuntime, ext)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlExt)
		}
	}

	return res, nil
}

func hybridMachineExtensionToMql(runtime *plugin.Runtime, ext *hybridcompute.MachineExtension) (*mqlAzureSubscriptionComputeServiceHybridMachineExtension, error) {
	var (
		publisher, extensionType, typeHandlerVersion, provisioningState, forceUpdateTag *string
		autoUpgradeMinorVersion, enableAutomaticUpgrade                                 *bool
		settingsDict, instanceViewDict                                                  map[string]any
	)
	if p := ext.Properties; p != nil {
		publisher = p.Publisher
		extensionType = p.Type
		typeHandlerVersion = p.TypeHandlerVersion
		provisioningState = p.ProvisioningState
		forceUpdateTag = p.ForceUpdateTag
		autoUpgradeMinorVersion = p.AutoUpgradeMinorVersion
		enableAutomaticUpgrade = p.EnableAutomaticUpgrade
		if p.Settings != nil {
			s, err := convert.JsonToDict(p.Settings)
			if err != nil {
				return nil, err
			}
			settingsDict = s
		}
		if p.InstanceView != nil {
			v, err := convert.JsonToDict(p.InstanceView)
			if err != nil {
				return nil, err
			}
			instanceViewDict = v
		}
	}
	systemData, err := convert.JsonToDict(ext.SystemData)
	if err != nil {
		return nil, err
	}
	mqlExt, err := CreateResource(runtime, "azure.subscription.computeService.hybridMachine.extension",
		map[string]*llx.RawData{
			"id":                      llx.StringDataPtr(ext.ID),
			"name":                    llx.StringDataPtr(ext.Name),
			"type":                    llx.StringDataPtr(ext.Type),
			"location":                llx.StringDataPtr(ext.Location),
			"tags":                    llx.MapData(convert.PtrMapStrToInterface(ext.Tags), types.String),
			"publisher":               llx.StringDataPtr(publisher),
			"extensionType":           llx.StringDataPtr(extensionType),
			"typeHandlerVersion":      llx.StringDataPtr(typeHandlerVersion),
			"autoUpgradeMinorVersion": llx.BoolDataPtr(autoUpgradeMinorVersion),
			"enableAutomaticUpgrade":  llx.BoolDataPtr(enableAutomaticUpgrade),
			"provisioningState":       llx.StringDataPtr(provisioningState),
			"settings":                llx.DictData(settingsDict),
			"forceUpdateTag":          llx.StringDataPtr(forceUpdateTag),
			"systemData":              llx.DictData(systemData),
			"instanceView":            llx.DictData(instanceViewDict),
		})
	if err != nil {
		return nil, err
	}
	return mqlExt.(*mqlAzureSubscriptionComputeServiceHybridMachineExtension), nil
}

func initAzureSubscriptionComputeServiceHybridMachine(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure hybrid machine")
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
	machines := computeSvc.GetHybridMachines()
	if machines.Error != nil {
		return nil, nil, machines.Error
	}
	id := args["id"].Value.(string)
	wantID := strings.ToLower(id)
	for _, entry := range machines.Data {
		machine := entry.(*mqlAzureSubscriptionComputeServiceHybridMachine)
		if strings.ToLower(machine.Id.Data) == wantID {
			return args, machine, nil
		}
	}

	return nil, nil, errors.New("azure hybrid machine does not exist")
}
