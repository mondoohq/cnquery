// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armbatch "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/azure/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func initAzureSubscriptionBatchService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionBatchService) id() (string, error) {
	return "azure.subscription.batch/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionBatchServiceAccount) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionBatchService) accounts() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	ctx := context.Background()
	client, err := armbatch.NewAccountClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(&armbatch.AccountClientListOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, entry := range page.Value {
			if entry == nil {
				continue
			}

			resource, err := batchAccountToMql(a.MqlRuntime, entry)
			if err != nil {
				return nil, err
			}
			res = append(res, resource)
		}
	}

	return res, nil
}

func batchAccountToMql(runtime *plugin.Runtime, account *armbatch.Account) (*mqlAzureSubscriptionBatchServiceAccount, error) {
	identityData := llx.NilData
	if account.Identity != nil {
		identity, err := convert.JsonToDict(account.Identity)
		if err != nil {
			return nil, err
		}
		identityData = llx.DictData(identity)
	}

	propertiesData := llx.NilData
	var (
		accountEndpoint                       = llx.NilData
		provisioningState                     = llx.NilData
		poolAllocationMode                    = llx.NilData
		publicNetworkAccess                   = llx.NilData
		nodeManagementEndpoint                = llx.NilData
		activeJobAndJobScheduleQuota          = llx.NilData
		dedicatedCoreQuota                    = llx.NilData
		dedicatedCoreQuotaPerVmFamilyEnforced = llx.NilData
		dedicatedCoreQuotaPerVmFamily         = llx.NilData
		lowPriorityCoreQuota                  = llx.NilData
		poolQuota                             = llx.NilData
		allowedAuthenticationModes            = llx.NilData
		autoStorage                           = llx.NilData
		encryption                            = llx.NilData
		keyVaultReference                     = llx.NilData
		networkProfile                        = llx.NilData
		privateEndpointConnections            = llx.NilData
	)

	if account.Properties != nil {
		props := account.Properties

		if dict, err := convert.JsonToDict(props); err != nil {
			return nil, err
		} else if dict != nil {
			propertiesData = llx.DictData(dict)
		}

		if props.AccountEndpoint != nil {
			accountEndpoint = llx.StringData(*props.AccountEndpoint)
		}
		if props.ProvisioningState != nil {
			provisioningState = llx.StringData(string(*props.ProvisioningState))
		}
		if props.PoolAllocationMode != nil {
			poolAllocationMode = llx.StringData(string(*props.PoolAllocationMode))
		}
		if props.PublicNetworkAccess != nil {
			publicNetworkAccess = llx.StringData(string(*props.PublicNetworkAccess))
		}
		if props.NodeManagementEndpoint != nil {
			nodeManagementEndpoint = llx.StringData(*props.NodeManagementEndpoint)
		}
		if props.ActiveJobAndJobScheduleQuota != nil {
			activeJobAndJobScheduleQuota = llx.IntData(int64(*props.ActiveJobAndJobScheduleQuota))
		}
		if props.DedicatedCoreQuota != nil {
			dedicatedCoreQuota = llx.IntData(int64(*props.DedicatedCoreQuota))
		}
		if props.DedicatedCoreQuotaPerVMFamilyEnforced != nil {
			dedicatedCoreQuotaPerVmFamilyEnforced = llx.BoolData(*props.DedicatedCoreQuotaPerVMFamilyEnforced)
		}
		if props.LowPriorityCoreQuota != nil {
			lowPriorityCoreQuota = llx.IntData(int64(*props.LowPriorityCoreQuota))
		}
		if props.PoolQuota != nil {
			poolQuota = llx.IntData(int64(*props.PoolQuota))
		}

		if props.AllowedAuthenticationModes != nil {
			values := []any{}
			for _, mode := range props.AllowedAuthenticationModes {
				if mode == nil {
					continue
				}
				values = append(values, string(*mode))
			}
			if values != nil {
				allowedAuthenticationModes = llx.ArrayData(values, types.String)
			}
		}

		if props.AutoStorage != nil {
			if dict, err := convert.JsonToDict(props.AutoStorage); err != nil {
				return nil, err
			} else if dict != nil {
				autoStorage = llx.DictData(dict)
			}
		}
		if props.Encryption != nil {
			if dict, err := convert.JsonToDict(props.Encryption); err != nil {
				return nil, err
			} else if dict != nil {
				encryption = llx.DictData(dict)
			}
		}
		if props.KeyVaultReference != nil {
			if dict, err := convert.JsonToDict(props.KeyVaultReference); err != nil {
				return nil, err
			} else if dict != nil {
				keyVaultReference = llx.DictData(dict)
			}
		}
		if props.NetworkProfile != nil {
			if dict, err := convert.JsonToDict(props.NetworkProfile); err != nil {
				return nil, err
			} else if dict != nil {
				networkProfile = llx.DictData(dict)
			}
		}

		if props.DedicatedCoreQuotaPerVMFamily != nil {
			items := []any{}
			for _, entry := range props.DedicatedCoreQuotaPerVMFamily {
				if entry == nil {
					continue
				}
				dict, err := convert.JsonToDict(entry)
				if err != nil {
					return nil, err
				}
				items = append(items, dict)
			}
			if items != nil {
				dedicatedCoreQuotaPerVmFamily = llx.ArrayData(items, types.Dict)
			}
		}

		if props.PrivateEndpointConnections != nil {
			items := []any{}
			for _, entry := range props.PrivateEndpointConnections {
				if entry == nil {
					continue
				}
				dict, err := convert.JsonToDict(entry)
				if err != nil {
					return nil, err
				}
				items = append(items, dict)
			}
			if items != nil {
				privateEndpointConnections = llx.ArrayData(items, types.Dict)
			}
		}
	}

	res, err := CreateResource(runtime, ResourceAzureSubscriptionBatchServiceAccount, map[string]*llx.RawData{
		"id":                                    llx.StringDataPtr(account.ID),
		"name":                                  llx.StringDataPtr(account.Name),
		"location":                              llx.StringDataPtr(account.Location),
		"tags":                                  llx.MapData(convert.PtrMapStrToInterface(account.Tags), types.String),
		"type":                                  llx.StringDataPtr(account.Type),
		"identity":                              identityData,
		"properties":                            propertiesData,
		"accountEndpoint":                       accountEndpoint,
		"provisioningState":                     provisioningState,
		"poolAllocationMode":                    poolAllocationMode,
		"publicNetworkAccess":                   publicNetworkAccess,
		"nodeManagementEndpoint":                nodeManagementEndpoint,
		"activeJobAndJobScheduleQuota":          activeJobAndJobScheduleQuota,
		"dedicatedCoreQuota":                    dedicatedCoreQuota,
		"dedicatedCoreQuotaPerVmFamilyEnforced": dedicatedCoreQuotaPerVmFamilyEnforced,
		"dedicatedCoreQuotaPerVmFamily":         dedicatedCoreQuotaPerVmFamily,
		"lowPriorityCoreQuota":                  lowPriorityCoreQuota,
		"poolQuota":                             poolQuota,
		"allowedAuthenticationModes":            allowedAuthenticationModes,
		"autoStorage":                           autoStorage,
		"encryption":                            encryption,
		"keyVaultReference":                     keyVaultReference,
		"networkProfile":                        networkProfile,
		"privateEndpointConnections":            privateEndpointConnections,
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlAzureSubscriptionBatchServiceAccount), nil
}
