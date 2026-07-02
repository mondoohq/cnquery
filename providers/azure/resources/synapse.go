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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/synapse/armsynapse"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlAzureSubscriptionSynapseServiceWorkspaceInternal struct {
	cacheDefaultStorageId           string
	cacheCmkKeyURI                  string
	cacheUserAssignedIdentityIds    []string
	cachePrivateEndpointConnections []*armsynapse.PrivateEndpointConnection
}

func (a *mqlAzureSubscriptionSynapseServiceWorkspace) privateEndpointConnections() ([]any, error) {
	return azurePrivateEndpointConnectionsToMql(a.MqlRuntime, a.cachePrivateEndpointConnections)
}

func (a *mqlAzureSubscriptionSynapseService) id() (string, error) {
	return "azure.subscription.synapse/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionSynapseService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionSynapseService) workspaces() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	subId := a.SubscriptionId.Data
	ctx := context.Background()

	client, err := armsynapse.NewWorkspacesClient(subId, conn.Token(), &arm.ClientOptions{
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
				log.Warn().Err(err).Msg("could not list synapse workspaces due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, ws := range page.Value {
			if ws == nil {
				continue
			}

			properties, err := convert.JsonToDict(ws.Properties)
			if err != nil {
				return nil, err
			}
			identity, err := convert.JsonToDict(ws.Identity)
			if err != nil {
				return nil, err
			}

			var managedVirtualNetwork string
			var publicNetworkAccess string
			var encryption any
			var managedResourceGroupName string
			var sqlAdministratorLogin string
			var provisioningState string
			var trustedServiceBypassEnabled *bool
			var azureADOnlyAuthentication *bool
			var defaultStorageFilesystem string
			var defaultStorageId string
			var cmkKeyURI string

			if ws.Properties != nil {
				if ws.Properties.ManagedVirtualNetwork != nil {
					managedVirtualNetwork = *ws.Properties.ManagedVirtualNetwork
				}
				if ws.Properties.PublicNetworkAccess != nil {
					publicNetworkAccess = string(*ws.Properties.PublicNetworkAccess)
				}
				if ws.Properties.Encryption != nil {
					encryption, err = convert.JsonToDict(ws.Properties.Encryption)
					if err != nil {
						return nil, err
					}
					if ws.Properties.Encryption.Cmk != nil && ws.Properties.Encryption.Cmk.Key != nil {
						cmkKeyURI = synapseKeyURI(ws.Properties.Encryption.Cmk.Key)
					}
				}
				if ws.Properties.DefaultDataLakeStorage != nil {
					if ws.Properties.DefaultDataLakeStorage.Filesystem != nil {
						defaultStorageFilesystem = *ws.Properties.DefaultDataLakeStorage.Filesystem
					}
					if ws.Properties.DefaultDataLakeStorage.ResourceID != nil {
						defaultStorageId = *ws.Properties.DefaultDataLakeStorage.ResourceID
					}
				}
				if ws.Properties.ManagedResourceGroupName != nil {
					managedResourceGroupName = *ws.Properties.ManagedResourceGroupName
				}
				if ws.Properties.SQLAdministratorLogin != nil {
					sqlAdministratorLogin = *ws.Properties.SQLAdministratorLogin
				}
				if ws.Properties.ProvisioningState != nil {
					provisioningState = *ws.Properties.ProvisioningState
				}
				trustedServiceBypassEnabled = ws.Properties.TrustedServiceBypassEnabled
				azureADOnlyAuthentication = ws.Properties.AzureADOnlyAuthentication
			}

			var userAssignedIdentityIds []string
			if ws.Identity != nil {
				userAssignedIdentityIds = sortedUserAssignedIdentityIDs(ws.Identity.UserAssignedIdentities)
			}

			mqlWorkspace, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionSynapseServiceWorkspace,
				map[string]*llx.RawData{
					"__id":                        llx.StringDataPtr(ws.ID),
					"id":                          llx.StringDataPtr(ws.ID),
					"name":                        llx.StringDataPtr(ws.Name),
					"location":                    llx.StringDataPtr(ws.Location),
					"tags":                        llx.MapData(convert.PtrMapStrToInterface(ws.Tags), types.String),
					"type":                        llx.StringDataPtr(ws.Type),
					"properties":                  llx.DictData(properties),
					"identity":                    llx.DictData(identity),
					"managedVirtualNetwork":       llx.StringData(managedVirtualNetwork),
					"publicNetworkAccess":         llx.StringData(publicNetworkAccess),
					"encryption":                  llx.DictData(encryption),
					"managedResourceGroupName":    llx.StringData(managedResourceGroupName),
					"sqlAdministratorLogin":       llx.StringData(sqlAdministratorLogin),
					"provisioningState":           llx.StringData(provisioningState),
					"trustedServiceBypassEnabled": llx.BoolDataPtr(trustedServiceBypassEnabled),
					"azureADOnlyAuthentication":   llx.BoolDataPtr(azureADOnlyAuthentication),
					"defaultStorageFilesystem":    llx.StringData(defaultStorageFilesystem),
				})
			if err != nil {
				return nil, err
			}
			workspaceRes := mqlWorkspace.(*mqlAzureSubscriptionSynapseServiceWorkspace)
			workspaceRes.cacheDefaultStorageId = defaultStorageId
			workspaceRes.cacheCmkKeyURI = cmkKeyURI
			workspaceRes.cacheUserAssignedIdentityIds = userAssignedIdentityIds
			if ws.Properties != nil {
				workspaceRes.cachePrivateEndpointConnections = ws.Properties.PrivateEndpointConnections
			}
			res = append(res, workspaceRes)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionSynapseServiceWorkspace) id() (string, error) {
	return a.Id.Data, nil
}

// synapseKeyURI normalizes the workspace CMK reference into a full Key Vault key
// identifier. The SDK's KeyVaultURL is documented as the full key identifier
// (https://{vault}.vault.azure.net/keys/{name}); if only a vault base URL is
// returned it is combined with the key name.
func synapseKeyURI(key *armsynapse.WorkspaceKeyDetails) string {
	if key == nil || key.KeyVaultURL == nil {
		return ""
	}
	url := strings.TrimSuffix(*key.KeyVaultURL, "/")
	if url == "" {
		return ""
	}
	if strings.Contains(url, "/keys/") {
		return url
	}
	if key.Name == nil || *key.Name == "" {
		return ""
	}
	return url + "/keys/" + *key.Name
}

// defaultStorage returns a typed reference to the workspace's default Data Lake Storage account.
func (a *mqlAzureSubscriptionSynapseServiceWorkspace) defaultStorage() (*mqlAzureSubscriptionStorageServiceAccount, error) {
	if a.cacheDefaultStorageId == "" {
		a.DefaultStorage.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "azure.subscription.storageService.account",
		map[string]*llx.RawData{"id": llx.StringData(a.cacheDefaultStorageId)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionStorageServiceAccount), nil
}

// cmkKey returns a typed reference to the Key Vault key used for customer-managed encryption.
func (a *mqlAzureSubscriptionSynapseServiceWorkspace) cmkKey() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	if a.cacheCmkKeyURI == "" {
		a.CmkKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return newKeyVaultKeyResource(a.MqlRuntime, a.cacheCmkKeyURI)
}

// userAssignedIdentities returns the typed user-assigned managed identities of the workspace.
func (a *mqlAzureSubscriptionSynapseServiceWorkspace) userAssignedIdentities() ([]any, error) {
	return resolveUserAssignedIdentities(a.MqlRuntime, a.cacheUserAssignedIdentityIds)
}

func initAzureSubscriptionSynapseServiceWorkspace(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure synapse workspace")
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.synapseService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	svc := res.(*mqlAzureSubscriptionSynapseService)
	list := svc.GetWorkspaces()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	id, ok := args["id"].Value.(string)
	if !ok {
		return nil, nil, errors.New("id must be a non-nil string value")
	}
	for _, entry := range list.Data {
		ws := entry.(*mqlAzureSubscriptionSynapseServiceWorkspace)
		if ws.Id.Data == id {
			return args, ws, nil
		}
	}

	return nil, nil, errors.New("azure synapse workspace does not exist")
}
