// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datafactory/armdatafactory/v10"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlAzureSubscriptionDataFactoryServiceFactoryInternal struct {
	cacheUserAssignedIdentityIds []string
}

func (a *mqlAzureSubscriptionDataFactoryService) id() (string, error) {
	return "azure.subscription.dataFactory/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionDataFactoryService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionDataFactoryService) factories() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	subId := a.SubscriptionId.Data
	ctx := context.Background()

	client, err := armdatafactory.NewFactoriesClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list data factories due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, factory := range page.Value {
			if factory == nil {
				continue
			}

			properties, err := convert.JsonToDict(factory.Properties)
			if err != nil {
				return nil, err
			}
			identity, err := convert.JsonToDict(factory.Identity)
			if err != nil {
				return nil, err
			}

			var userAssignedIdentityIds []string
			if factory.Identity != nil {
				userAssignedIdentityIds = sortedUserAssignedIdentityIDs(factory.Identity.UserAssignedIdentities)
			}

			var publicNetworkAccess string
			var provisioningState string
			var version string
			var repoConfig any
			var encryption any
			var cmkKeyName, cmkKeyVaultUri, cmkKeyVersion, cmkUserAssignedIdentity string
			var created *llx.RawData

			if factory.Properties != nil {
				if factory.Properties.PublicNetworkAccess != nil {
					publicNetworkAccess = string(*factory.Properties.PublicNetworkAccess)
				}
				if factory.Properties.ProvisioningState != nil {
					provisioningState = *factory.Properties.ProvisioningState
				}
				if factory.Properties.Version != nil {
					version = *factory.Properties.Version
				}
				if factory.Properties.RepoConfiguration != nil {
					repoConfig, err = convert.JsonToDict(factory.Properties.RepoConfiguration)
					if err != nil {
						return nil, err
					}
				}
				if factory.Properties.Encryption != nil {
					encryption, err = convert.JsonToDict(factory.Properties.Encryption)
					if err != nil {
						return nil, err
					}
					if factory.Properties.Encryption.KeyName != nil {
						cmkKeyName = *factory.Properties.Encryption.KeyName
					}
					if factory.Properties.Encryption.VaultBaseURL != nil {
						cmkKeyVaultUri = *factory.Properties.Encryption.VaultBaseURL
					}
					if factory.Properties.Encryption.KeyVersion != nil {
						cmkKeyVersion = *factory.Properties.Encryption.KeyVersion
					}
					if factory.Properties.Encryption.Identity != nil && factory.Properties.Encryption.Identity.UserAssignedIdentity != nil {
						cmkUserAssignedIdentity = *factory.Properties.Encryption.Identity.UserAssignedIdentity
					}
				}
				created = llx.TimeDataPtr(factory.Properties.CreateTime)
			}
			if created == nil {
				created = llx.TimeData(llx.NeverFutureTime)
			}

			mqlFactory, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionDataFactoryServiceFactory,
				map[string]*llx.RawData{
					"__id":                    llx.StringDataPtr(factory.ID),
					"id":                      llx.StringDataPtr(factory.ID),
					"name":                    llx.StringDataPtr(factory.Name),
					"location":                llx.StringDataPtr(factory.Location),
					"tags":                    llx.MapData(convert.PtrMapStrToInterface(factory.Tags), types.String),
					"type":                    llx.StringDataPtr(factory.Type),
					"properties":              llx.DictData(properties),
					"publicNetworkAccess":     llx.StringData(publicNetworkAccess),
					"identity":                llx.DictData(identity),
					"provisioningState":       llx.StringData(provisioningState),
					"version":                 llx.StringData(version),
					"repoConfiguration":       llx.DictData(repoConfig),
					"encryption":              llx.DictData(encryption),
					"cmkKeyName":              llx.StringData(cmkKeyName),
					"cmkKeyVaultUri":          llx.StringData(cmkKeyVaultUri),
					"cmkKeyVersion":           llx.StringData(cmkKeyVersion),
					"cmkUserAssignedIdentity": llx.StringData(cmkUserAssignedIdentity),
					"created":                 created,
				})
			if err != nil {
				return nil, err
			}
			factoryRes := mqlFactory.(*mqlAzureSubscriptionDataFactoryServiceFactory)
			factoryRes.cacheUserAssignedIdentityIds = userAssignedIdentityIds
			res = append(res, factoryRes)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionDataFactoryServiceFactory) id() (string, error) {
	return a.Id.Data, nil
}

// cmkKey returns a typed reference to the Key Vault key used for customer-managed encryption.
func (a *mqlAzureSubscriptionDataFactoryServiceFactory) cmkKey() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	vaultURI := strings.TrimSuffix(a.CmkKeyVaultUri.Data, "/")
	keyName := a.CmkKeyName.Data
	if vaultURI == "" || keyName == "" {
		a.CmkKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	keyURI := vaultURI + "/keys/" + keyName
	if version := a.CmkKeyVersion.Data; version != "" {
		keyURI += "/" + version
	}
	return newKeyVaultKeyResource(a.MqlRuntime, keyURI)
}

// cmkIdentity returns the user-assigned managed identity used to access the CMK.
func (a *mqlAzureSubscriptionDataFactoryServiceFactory) cmkIdentity() (*mqlAzureSubscriptionManagedIdentity, error) {
	id := a.CmkUserAssignedIdentity.Data
	if id == "" {
		a.CmkIdentity.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "azure.subscription.managedIdentity",
		map[string]*llx.RawData{"__id": llx.StringData(id)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionManagedIdentity), nil
}

// userAssignedIdentities returns the typed user-assigned managed identities of the factory.
func (a *mqlAzureSubscriptionDataFactoryServiceFactory) userAssignedIdentities() ([]any, error) {
	return resolveUserAssignedIdentities(a.MqlRuntime, a.cacheUserAssignedIdentityIds)
}

func initAzureSubscriptionDataFactoryServiceFactory(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure data factory")
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.dataFactoryService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	svc := res.(*mqlAzureSubscriptionDataFactoryService)
	list := svc.GetFactories()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	id, ok := args["id"].Value.(string)
	if !ok {
		return nil, nil, errors.New("id must be a non-nil string value")
	}
	for _, entry := range list.Data {
		factory := entry.(*mqlAzureSubscriptionDataFactoryServiceFactory)
		if factory.Id.Data == id {
			return args, factory, nil
		}
	}

	return nil, nil, errors.New("azure data factory does not exist")
}
