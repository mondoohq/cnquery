// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/databricks/armdatabricks"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func initAzureSubscriptionDatabricksService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionDatabricksService) id() (string, error) {
	return "azure.subscription.databricksService/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionDatabricksServiceWorkspace) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionDatabricksService) workspaces() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided. it is not an Azure connection")
	}

	ctx := context.Background()
	client, err := armdatabricks.NewWorkspacesClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionPager(&armdatabricks.WorkspacesClientListBySubscriptionOptions{})
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, entry := range page.Value {
			if entry == nil {
				continue
			}

			resource, err := databricksWorkspaceToMql(a.MqlRuntime, entry)
			if err != nil {
				return nil, err
			}
			res = append(res, resource)
		}
	}

	return res, nil
}

func databricksWorkspaceToMql(runtime *plugin.Runtime, workspace *armdatabricks.Workspace) (*mqlAzureSubscriptionDatabricksServiceWorkspace, error) {
	propertiesData := llx.NilData
	if workspace.Properties != nil {
		if dict, err := convert.JsonToDict(workspace.Properties); err != nil {
			return nil, err
		} else if dict != nil {
			propertiesData = llx.DictData(dict)
		}
	}

	skuData := llx.NilData
	if workspace.SKU != nil {
		if dict, err := convert.JsonToDict(workspace.SKU); err != nil {
			return nil, err
		} else if dict != nil {
			skuData = llx.DictData(dict)
		}
	}

	var publicNetworkAccess, requiredNsgRules, diskEncryptionSetId, managedResourceGroupId, provisioningState, workspaceId string
	var enableNoPublicIp, requireInfraEnc bool
	var customVnetId string
	var managedDiskKeySource, managedDiskKeyVaultUri, managedDiskKeyName, managedDiskKeyVersion string
	var managedServicesKeySource, managedServicesKeyVaultUri, managedServicesKeyName, managedServicesKeyVersion string

	if props := workspace.Properties; props != nil {
		if props.PublicNetworkAccess != nil {
			publicNetworkAccess = string(*props.PublicNetworkAccess)
		}
		if props.RequiredNsgRules != nil {
			requiredNsgRules = string(*props.RequiredNsgRules)
		}
		if props.DiskEncryptionSetID != nil {
			diskEncryptionSetId = *props.DiskEncryptionSetID
		}
		if props.ManagedResourceGroupID != nil {
			managedResourceGroupId = *props.ManagedResourceGroupID
		}
		if props.ProvisioningState != nil {
			provisioningState = string(*props.ProvisioningState)
		}
		if props.WorkspaceID != nil {
			workspaceId = *props.WorkspaceID
		}

		if p := props.Parameters; p != nil {
			if p.EnableNoPublicIP != nil && p.EnableNoPublicIP.Value != nil {
				enableNoPublicIp = *p.EnableNoPublicIP.Value
			}
			if p.RequireInfrastructureEncryption != nil && p.RequireInfrastructureEncryption.Value != nil {
				requireInfraEnc = *p.RequireInfrastructureEncryption.Value
			}
			if p.CustomVirtualNetworkID != nil && p.CustomVirtualNetworkID.Value != nil {
				customVnetId = *p.CustomVirtualNetworkID.Value
			}
			if p.Encryption != nil && p.Encryption.Value != nil {
				if p.Encryption.Value.KeySource != nil {
					managedDiskKeySource = string(*p.Encryption.Value.KeySource)
				}
				if p.Encryption.Value.KeyVaultURI != nil {
					managedDiskKeyVaultUri = *p.Encryption.Value.KeyVaultURI
				}
				if p.Encryption.Value.KeyName != nil {
					managedDiskKeyName = *p.Encryption.Value.KeyName
				}
				if p.Encryption.Value.KeyVersion != nil {
					managedDiskKeyVersion = *p.Encryption.Value.KeyVersion
				}
			}
		}

		if enc := props.Encryption; enc != nil && enc.Entities != nil {
			if md := enc.Entities.ManagedDisk; md != nil {
				// KeySource may be "Default" (Microsoft-managed) without KeyVaultProperties,
				// or "Microsoft.Keyvault" with KV details. Read it independently.
				if md.KeySource != nil {
					managedDiskKeySource = string(*md.KeySource)
				}
				if md.KeyVaultProperties != nil {
					if md.KeyVaultProperties.KeyVaultURI != nil {
						managedDiskKeyVaultUri = *md.KeyVaultProperties.KeyVaultURI
					}
					if md.KeyVaultProperties.KeyName != nil {
						managedDiskKeyName = *md.KeyVaultProperties.KeyName
					}
					if md.KeyVaultProperties.KeyVersion != nil {
						managedDiskKeyVersion = *md.KeyVaultProperties.KeyVersion
					}
				}
			}
			if ms := enc.Entities.ManagedServices; ms != nil {
				if ms.KeySource != nil {
					managedServicesKeySource = string(*ms.KeySource)
				}
				if ms.KeyVaultProperties != nil {
					if ms.KeyVaultProperties.KeyVaultURI != nil {
						managedServicesKeyVaultUri = *ms.KeyVaultProperties.KeyVaultURI
					}
					if ms.KeyVaultProperties.KeyName != nil {
						managedServicesKeyName = *ms.KeyVaultProperties.KeyName
					}
					if ms.KeyVaultProperties.KeyVersion != nil {
						managedServicesKeyVersion = *ms.KeyVaultProperties.KeyVersion
					}
				}
			}
		}
	}

	res, err := CreateResource(runtime, ResourceAzureSubscriptionDatabricksServiceWorkspace, map[string]*llx.RawData{
		"id":                              llx.StringDataPtr(workspace.ID),
		"name":                            llx.StringDataPtr(workspace.Name),
		"location":                        llx.StringDataPtr(workspace.Location),
		"tags":                            llx.MapData(convert.PtrMapStrToInterface(workspace.Tags), types.String),
		"type":                            llx.StringDataPtr(workspace.Type),
		"properties":                      propertiesData,
		"sku":                             skuData,
		"publicNetworkAccess":             llx.StringData(publicNetworkAccess),
		"enableNoPublicIp":                llx.BoolData(enableNoPublicIp),
		"requireInfrastructureEncryption": llx.BoolData(requireInfraEnc),
		"customVirtualNetworkId":          llx.StringData(customVnetId),
		"requiredNsgRules":                llx.StringData(requiredNsgRules),
		"diskEncryptionSetId":             llx.StringData(diskEncryptionSetId),
		"managedResourceGroupId":          llx.StringData(managedResourceGroupId),
		"provisioningState":               llx.StringData(provisioningState),
		"workspaceId":                     llx.StringData(workspaceId),
		"managedDiskKeySource":            llx.StringData(managedDiskKeySource),
		"managedDiskKeyVaultUri":          llx.StringData(managedDiskKeyVaultUri),
		"managedDiskKeyName":              llx.StringData(managedDiskKeyName),
		"managedDiskKeyVersion":           llx.StringData(managedDiskKeyVersion),
		"managedServicesKeySource":        llx.StringData(managedServicesKeySource),
		"managedServicesKeyVaultUri":      llx.StringData(managedServicesKeyVaultUri),
		"managedServicesKeyName":          llx.StringData(managedServicesKeyName),
		"managedServicesKeyVersion":       llx.StringData(managedServicesKeyVersion),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlAzureSubscriptionDatabricksServiceWorkspace), nil
}
