// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	ml "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/machinelearning/armmachinelearning/v4"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlAzureSubscriptionMachineLearningServiceWorkspaceInternal struct {
	cacheEncryptionKeyUri string
}

func initAzureSubscriptionMachineLearningService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionMachineLearningService) id() (string, error) {
	return "azure.subscription.machineLearningService/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionMachineLearningServiceWorkspace) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMachineLearningService) workspaces() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	ctx := context.Background()
	client, err := ml.NewWorkspacesClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
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
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list machine learning workspaces due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, ws := range page.Value {
			if ws == nil {
				continue
			}
			mqlWs, err := machineLearningWorkspaceToMql(a.MqlRuntime, ws)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlWs)
		}
	}
	return res, nil
}

func machineLearningWorkspaceToMql(runtime *plugin.Runtime, ws *ml.Workspace) (*mqlAzureSubscriptionMachineLearningServiceWorkspace, error) {
	sku, err := convert.JsonToDict(ws.SKU)
	if err != nil {
		return nil, err
	}
	identity, err := convert.JsonToDict(ws.Identity)
	if err != nil {
		return nil, err
	}

	props := ws.Properties
	if props == nil {
		props = &ml.WorkspaceProperties{}
	}

	var publicNetworkAccess, provisioningState, managedNetworkIsolationMode string
	if props.PublicNetworkAccess != nil {
		publicNetworkAccess = string(*props.PublicNetworkAccess)
	}
	if props.ProvisioningState != nil {
		provisioningState = string(*props.ProvisioningState)
	}
	if props.ManagedNetwork != nil && props.ManagedNetwork.IsolationMode != nil {
		managedNetworkIsolationMode = string(*props.ManagedNetwork.IsolationMode)
	}

	var encryptionStatus, encryptionKeyUri string
	if props.Encryption != nil {
		if props.Encryption.Status != nil {
			encryptionStatus = string(*props.Encryption.Status)
		}
		if props.Encryption.KeyVaultProperties != nil {
			encryptionKeyUri = convert.ToValue(props.Encryption.KeyVaultProperties.KeyIdentifier)
		}
	}

	peConns, err := convert.JsonToDictSlice(props.PrivateEndpointConnections)
	if err != nil {
		return nil, err
	}

	resource, err := CreateResource(runtime, ResourceAzureSubscriptionMachineLearningServiceWorkspace,
		map[string]*llx.RawData{
			"id":                              llx.StringDataPtr(ws.ID),
			"name":                            llx.StringDataPtr(ws.Name),
			"location":                        llx.StringDataPtr(ws.Location),
			"tags":                            llx.MapData(convert.PtrMapStrToInterface(ws.Tags), types.String),
			"kind":                            llx.StringDataPtr(ws.Kind),
			"sku":                             llx.DictData(sku),
			"identity":                        llx.DictData(identity),
			"workspaceId":                     llx.StringDataPtr(props.WorkspaceID),
			"friendlyName":                    llx.StringDataPtr(props.FriendlyName),
			"description":                     llx.StringDataPtr(props.Description),
			"provisioningState":               llx.StringData(provisioningState),
			"publicNetworkAccess":             llx.StringData(publicNetworkAccess),
			"managedNetworkIsolationMode":     llx.StringData(managedNetworkIsolationMode),
			"hbiWorkspace":                    llx.BoolDataPtr(props.HbiWorkspace),
			"allowPublicAccessWhenBehindVnet": llx.BoolDataPtr(props.AllowPublicAccessWhenBehindVnet),
			"v1LegacyMode":                    llx.BoolDataPtr(props.V1LegacyMode),
			"discoveryUrl":                    llx.StringDataPtr(props.DiscoveryURL),
			"mlFlowTrackingUri":               llx.StringDataPtr(props.MlFlowTrackingURI),
			"imageBuildCompute":               llx.StringDataPtr(props.ImageBuildCompute),
			"encryptionStatus":                llx.StringData(encryptionStatus),
			"privateEndpointConnections":      llx.ArrayData(peConns, types.Dict),
			"keyVaultId":                      llx.StringDataPtr(props.KeyVault),
			"storageAccountId":                llx.StringDataPtr(props.StorageAccount),
			"applicationInsightsId":           llx.StringDataPtr(props.ApplicationInsights),
			"containerRegistryId":             llx.StringDataPtr(props.ContainerRegistry),
		})
	if err != nil {
		return nil, err
	}

	mqlWs := resource.(*mqlAzureSubscriptionMachineLearningServiceWorkspace)
	mqlWs.cacheEncryptionKeyUri = encryptionKeyUri
	return mqlWs, nil
}

func (a *mqlAzureSubscriptionMachineLearningServiceWorkspace) encryptionKey() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	if a.cacheEncryptionKeyUri == "" {
		a.EncryptionKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return newKeyVaultKeyResource(a.MqlRuntime, a.cacheEncryptionKeyUri)
}

func (a *mqlAzureSubscriptionMachineLearningServiceWorkspace) keyVault() (*mqlAzureSubscriptionKeyVaultServiceVault, error) {
	if a.KeyVaultId.Data == "" {
		a.KeyVault.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionKeyVaultServiceVault,
		map[string]*llx.RawData{"id": llx.StringData(a.KeyVaultId.Data)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionKeyVaultServiceVault), nil
}

func (a *mqlAzureSubscriptionMachineLearningServiceWorkspace) storageAccount() (*mqlAzureSubscriptionStorageServiceAccount, error) {
	if a.StorageAccountId.Data == "" {
		a.StorageAccount.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	return getStorageAccount(a.StorageAccountId.Data, a.MqlRuntime, conn)
}

func (a *mqlAzureSubscriptionMachineLearningServiceWorkspace) applicationInsights() (*mqlAzureSubscriptionMonitorServiceApplicationInsight, error) {
	if a.ApplicationInsightsId.Data == "" {
		a.ApplicationInsights.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionMonitorServiceApplicationInsight,
		map[string]*llx.RawData{"id": llx.StringData(a.ApplicationInsightsId.Data)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionMonitorServiceApplicationInsight), nil
}

func (a *mqlAzureSubscriptionMachineLearningServiceWorkspace) containerRegistry() (*mqlAzureSubscriptionContainerRegistryServiceRegistry, error) {
	if a.ContainerRegistryId.Data == "" {
		a.ContainerRegistry.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionContainerRegistryServiceRegistry,
		map[string]*llx.RawData{"id": llx.StringData(a.ContainerRegistryId.Data)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionContainerRegistryServiceRegistry), nil
}
