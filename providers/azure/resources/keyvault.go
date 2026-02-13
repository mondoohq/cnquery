// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	keyvault "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

var keyvaultidRegex = regexp.MustCompile(`^(https:\/\/([^\/]*)\.vault\.azure\.net)\/(certificates|secrets|keys)\/([^\/]*)(?:\/([^\/]*)){0,1}$`)

type keyvaultid struct {
	BaseUrl string
	Vault   string
	Type    string
	Name    string
	Version string
}

func parseKeyVaultId(url string) (*keyvaultid, error) {
	m := keyvaultidRegex.FindStringSubmatch(url)

	if len(m) != 6 {
		return nil, fmt.Errorf("cannot parse azure keyvault id: %s", url)
	}

	return &keyvaultid{
		BaseUrl: m[1],
		Vault:   m[2],
		Type:    m[3],
		Name:    m[4],
		Version: m[5],
	}, nil
}

func (a *mqlAzureSubscriptionKeyVaultService) id() (string, error) {
	return "azure.subscription.keyVault/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionKeyVaultService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionKeyVaultServiceVault) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceKey) id() (string, error) {
	return a.Kid.Data, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceSecret) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceCertificate) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionKeyVaultService) vaults() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := keyvault.NewVaultsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(&keyvault.VaultsClientListOptions{})
	res := []any{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.keyVaultService.vault",
				map[string]*llx.RawData{
					"id":        llx.StringDataPtr(entry.ID),
					"vaultName": llx.StringDataPtr(entry.Name),
					"location":  llx.StringDataPtr(entry.Location),
					"type":      llx.StringDataPtr(entry.Type),
					"tags":      llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceVault) vaultUri() (string, error) {
	name := a.VaultName.Data
	KVUri := "https://" + name + ".vault.azure.net"
	return KVUri, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceVault) properties() (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	vaultName, err := resourceID.Component("vaults")
	if err != nil {
		return nil, err
	}
	client, err := keyvault.NewVaultsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	vault, err := client.Get(ctx, resourceID.ResourceGroup, vaultName, &keyvault.VaultsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(vault.Properties)
}

func (a *mqlAzureSubscriptionKeyVaultServiceVault) rbacAuthorizationEnabled() (bool, error) {
	props := a.GetProperties()
	if props.Error != nil {
		return false, props.Error
	}
	propsDict := props.Data.(map[string]any)
	rbacProp := propsDict["enableRbacAuthorization"]
	if rbacProp == nil {
		return false, errors.New("key vault does not have enableRbacAuthorization property")
	}
	return rbacProp.(bool), nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceVault) keys() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	vaultUri := a.GetVaultUri()
	client, err := azkeys.NewClient(vaultUri.Data, token, &azkeys.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListKeyPropertiesPager(&azkeys.ListKeyPropertiesOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, entry := range page.Value {
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.keyVaultService.key",
				map[string]*llx.RawData{
					"kid":           llx.StringDataPtr((*string)(entry.KID)),
					"managed":       llx.BoolDataPtr(entry.Managed),
					"tags":          llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"enabled":       llx.BoolDataPtr(entry.Attributes.Enabled),
					"created":       llx.TimeDataPtr(entry.Attributes.Created),
					"updated":       llx.TimeDataPtr(entry.Attributes.Updated),
					"expires":       llx.TimeDataPtr(entry.Attributes.Expires),
					"notBefore":     llx.TimeDataPtr(entry.Attributes.NotBefore),
					"recoveryLevel": llx.StringDataPtr((*string)(entry.Attributes.RecoveryLevel)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceKeyAutorotation) id() (string, error) {
	id := a.Kid.Data
	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return "", err
	}

	return kvid.Name, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceVault) autorotation() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	vaultUri := a.GetVaultUri()
	client, err := azkeys.NewClient(vaultUri.Data, token, &azkeys.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListKeyPropertiesPager(&azkeys.ListKeyPropertiesOptions{})
	res := []any{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, entry := range page.Value {
			autoRotationEnabled := false

			if entry.KID != nil {
				keyID := string(*entry.KID)
				kvid, err := parseKeyVaultId(keyID)
				if err == nil && kvid.Type == "keys" {
					policyResp, err := client.GetKeyRotationPolicy(ctx, kvid.Name, nil)
					if err == nil && policyResp.LifetimeActions != nil {
						for _, action := range policyResp.LifetimeActions {
							if action.Action != nil && string(*action.Action.Type) == "Rotate" {
								autoRotationEnabled = true
								break
							}
						}
					}
				}
			}

			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.keyVaultService.key.autorotation",
				map[string]*llx.RawData{
					"kid":     llx.StringDataPtr((*string)(entry.KID)),
					"enabled": llx.BoolData(autoRotationEnabled),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceVault) secrets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	vaultUri := a.GetVaultUri()
	client, err := azsecrets.NewClient(vaultUri.Data, token, &azsecrets.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListSecretPropertiesPager(&azsecrets.ListSecretPropertiesOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, entry := range page.Value {
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.keyVaultService.secret",
				map[string]*llx.RawData{
					"id":          llx.StringDataPtr((*string)(entry.ID)),
					"tags":        llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"contentType": llx.StringDataPtr(entry.ContentType),
					"managed":     llx.BoolDataPtr(entry.Managed),
					"enabled":     llx.BoolDataPtr(entry.Attributes.Enabled),
					"created":     llx.TimeDataPtr(entry.Attributes.Created),
					"updated":     llx.TimeDataPtr(entry.Attributes.Updated),
					"expires":     llx.TimeDataPtr(entry.Attributes.Expires),
					"notBefore":   llx.TimeDataPtr(entry.Attributes.NotBefore),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceVault) certificates() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	vaultUri := a.GetVaultUri()
	client, err := azcertificates.NewClient(vaultUri.Data, token, &azcertificates.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListCertificatePropertiesPager(&azcertificates.ListCertificatePropertiesOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, entry := range page.Value {
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.keyVaultService.certificate",
				map[string]*llx.RawData{
					"id":            llx.StringDataPtr((*string)(entry.ID)),
					"tags":          llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"enabled":       llx.BoolDataPtr(entry.Attributes.Enabled),
					"created":       llx.TimeDataPtr(entry.Attributes.Created),
					"updated":       llx.TimeDataPtr(entry.Attributes.Updated),
					"expires":       llx.TimeDataPtr(entry.Attributes.Expires),
					"notBefore":     llx.TimeDataPtr(entry.Attributes.NotBefore),
					"recoveryLevel": llx.StringDataPtr((*string)(entry.Attributes.RecoveryLevel)),
					"x5t":           llx.StringData(hex.EncodeToString(entry.X509Thumbprint)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceVault) diagnosticSettings() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	return getDiagnosticSettings(a.Id.Data, a.MqlRuntime, conn)
}

func (a *mqlAzureSubscriptionKeyVaultServiceVault) privateEndpointConnections() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	vaultName, err := resourceID.Component("vaults")
	if err != nil {
		return nil, err
	}
	client, err := keyvault.NewVaultsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{ClientOptions: conn.ClientOptions()})
	if err != nil {
		return nil, err
	}

	vault, err := client.Get(ctx, resourceID.ResourceGroup, vaultName, &keyvault.VaultsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	var res []any
	if vault.Properties == nil || vault.Properties.PrivateEndpointConnections == nil {
		return res, nil
	}

	for _, entry := range vault.Properties.PrivateEndpointConnections {
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
				// Construct type from provider and path components
				if connResourceID.Provider != "" {
					resType = connResourceID.Provider + "/vaults/privateEndpointConnections"
				}
			}
			// Fallback: extract name from ID if Component fails
			if name == "" && entry.ID != nil {
				parts := strings.Split(*entry.ID, "/")
				if len(parts) > 0 {
					name = parts[len(parts)-1]
				}
			}
		}
		if resType == "" {
			resType = "Microsoft.KeyVault/vaults/privateEndpointConnections"
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
					stateArgs["actionsRequired"] = llx.StringData(string(*props.PrivateLinkServiceConnectionState.ActionsRequired))
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

func (a *mqlAzureSubscriptionKeyVaultServiceKey) keyName() (string, error) {
	id := a.Kid.Data
	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return "", err
	}

	return kvid.Name, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceKey) version() (string, error) {
	id := a.Kid.Data
	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return "", err
	}

	return kvid.Version, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceKey) versions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	id := a.Kid.Data
	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}

	if len(kvid.Version) > 0 {
		return nil, errors.New("cannot fetch versions for an already versioned azure key")
	}
	if kvid.Type != "keys" {
		return nil, errors.New("only key ids are supported")
	}

	client, err := azkeys.NewClient(kvid.BaseUrl, conn.Token(), &azkeys.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	pager := client.NewListKeyPropertiesVersionsPager(kvid.Name, &azkeys.ListKeyPropertiesVersionsOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.keyVaultService.key",
				map[string]*llx.RawData{
					"kid":           llx.StringDataPtr((*string)(entry.KID)),
					"managed":       llx.BoolDataPtr(entry.Managed),
					"tags":          llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"enabled":       llx.BoolDataPtr(entry.Attributes.Enabled),
					"created":       llx.TimeDataPtr(entry.Attributes.Created),
					"updated":       llx.TimeDataPtr(entry.Attributes.Updated),
					"expires":       llx.TimeDataPtr(entry.Attributes.Expires),
					"notBefore":     llx.TimeDataPtr(entry.Attributes.NotBefore),
					"recoveryLevel": llx.StringDataPtr((*string)(entry.Attributes.RecoveryLevel)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceKey) rotationPolicy() (*mqlAzureSubscriptionKeyVaultServiceKeyRotationPolicyObject, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	id := a.Kid.Data
	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}

	if kvid.Type != "keys" {
		return nil, errors.New("only key ids are supported")
	}

	client, err := azkeys.NewClient(kvid.BaseUrl, conn.Token(), &azkeys.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	policyResp, err := client.GetKeyRotationPolicy(ctx, kvid.Name, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			// Rotation policy doesn't exist, return a resource with enabled=false
			resource, err := CreateResource(a.MqlRuntime,
				ResourceAzureSubscriptionKeyVaultServiceKeyRotationPolicyObject,
				map[string]*llx.RawData{
					"__id":            llx.StringData(id + "/rotationPolicy"),
					"lifetimeActions": llx.ArrayData([]any{}, types.Dict),
					"attributes":      llx.DictData(map[string]any{}),
					"enabled":         llx.BoolData(false),
				},
			)
			if err != nil {
				return nil, err
			}
			return resource.(*mqlAzureSubscriptionKeyVaultServiceKeyRotationPolicyObject), nil
		}
		return nil, err
	}

	lifetimeActions := []any{}
	rotationEnabled := false
	if policyResp.LifetimeActions != nil {
		for _, action := range policyResp.LifetimeActions {
			actionDict, err := convert.JsonToDict(action)
			if err != nil {
				return nil, err
			}
			lifetimeActions = append(lifetimeActions, actionDict)

			if action.Action != nil && string(*action.Action.Type) == "Rotate" {
				rotationEnabled = true
			}
		}
	}

	attributes := map[string]any{}
	if policyResp.Attributes != nil {
		attributesDict, err := convert.JsonToDict(policyResp.Attributes)
		if err != nil {
			return nil, err
		}
		attributes = attributesDict
	}

	resource, err := CreateResource(a.MqlRuntime,
		ResourceAzureSubscriptionKeyVaultServiceKeyRotationPolicyObject,
		map[string]*llx.RawData{
			"__id":            llx.StringData(id + "/rotationPolicy"),
			"lifetimeActions": llx.ArrayData(lifetimeActions, types.Dict),
			"attributes":      llx.DictData(attributes),
			"enabled":         llx.BoolData(rotationEnabled),
		},
	)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionKeyVaultServiceKeyRotationPolicyObject), nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceCertificate) certName() (string, error) {
	id := a.Id.Data
	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return "", err
	}

	return kvid.Name, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceCertificate) version() (string, error) {
	id := a.Id.Data
	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return "", err
	}

	return kvid.Version, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceCertificate) versions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	id := a.Id.Data
	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}

	if len(kvid.Version) > 0 {
		return nil, errors.New("cannot fetch versions for an already versioned azure certificate")
	}
	if kvid.Type != "certificates" {
		return nil, errors.New("only certificate ids are supported")
	}

	vaultUrl := kvid.BaseUrl
	name := kvid.Name
	client, err := azcertificates.NewClient(vaultUrl, conn.Token(), &azcertificates.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	pager := client.NewListCertificatePropertiesVersionsPager(name, &azcertificates.ListCertificatePropertiesVersionsOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.keyVaultService.certificate",
				map[string]*llx.RawData{
					"id":            llx.StringDataPtr((*string)(entry.ID)),
					"tags":          llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"enabled":       llx.BoolDataPtr(entry.Attributes.Enabled),
					"created":       llx.TimeDataPtr(entry.Attributes.Created),
					"updated":       llx.TimeDataPtr(entry.Attributes.Updated),
					"expires":       llx.TimeDataPtr(entry.Attributes.Expires),
					"notBefore":     llx.TimeDataPtr(entry.Attributes.NotBefore),
					"recoveryLevel": llx.StringDataPtr((*string)(entry.Attributes.RecoveryLevel)),
					"x5t":           llx.StringData(hex.EncodeToString(entry.X509Thumbprint)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceCertificate) policy() (*mqlAzureSubscriptionKeyVaultServiceCertificatePolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	id := a.Id.Data
	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}

	if kvid.Type != "certificates" {
		return nil, errors.New("only certificate ids are supported")
	}

	client, err := azcertificates.NewClient(kvid.BaseUrl, conn.Token(), &azcertificates.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	policyResp, err := client.GetCertificatePolicy(ctx, kvid.Name, nil)
	if err != nil {
		// Only treat 404 (not found) as "policy doesn't exist"
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			// Certificate policy doesn't exist, return empty resource
			x509Props, err := CreateResource(a.MqlRuntime,
				"azure.subscription.keyVaultService.certificate.policy.x509CertificateProperties",
				map[string]*llx.RawData{
					"__id":             llx.StringData(id + "/policy/x509CertificateProperties"),
					"subject":          llx.StringData(""),
					"validityInMonths": llx.IntData(0),
					"keyUsage":         llx.ArrayData([]any{}, types.String),
					"ekus":             llx.ArrayData([]any{}, types.String),
				},
			)
			if err != nil {
				return nil, err
			}

			// Create empty key properties resource
			keyProps, err := CreateResource(a.MqlRuntime,
				"azure.subscription.keyVaultService.certificate.policy.keyProperties",
				map[string]*llx.RawData{
					"__id":       llx.StringData(id + "/policy/keyProperties"),
					"curve":      llx.StringData(""),
					"exportable": llx.BoolData(false),
					"keySize":    llx.IntData(0),
					"keyType":    llx.StringData(""),
					"reuseKey":   llx.BoolData(false),
				},
			)
			if err != nil {
				return nil, err
			}

			// Create empty issuer parameters resource
			issuerParams, err := CreateResource(a.MqlRuntime,
				"azure.subscription.keyVaultService.certificate.policy.issuerParameters",
				map[string]*llx.RawData{
					"__id":                    llx.StringData(id + "/policy/issuerParameters"),
					"certificateTransparency": llx.BoolData(false),
					"certificateType":         llx.StringData(""),
					"name":                    llx.StringData(""),
				},
			)
			if err != nil {
				return nil, err
			}

			resource, err := CreateResource(a.MqlRuntime,
				"azure.subscription.keyVaultService.certificate.policy",
				map[string]*llx.RawData{
					"__id":                      llx.StringData(id + "/policy"),
					"x509CertificateProperties": llx.ResourceData(x509Props, "azure.subscription.keyVaultService.certificate.policy.x509CertificateProperties"),
					"keyProperties":             llx.ResourceData(keyProps, "azure.subscription.keyVaultService.certificate.policy.keyProperties"),
					"issuerParameters":          llx.ResourceData(issuerParams, "azure.subscription.keyVaultService.certificate.policy.issuerParameters"),
				},
			)
			if err != nil {
				return nil, err
			}
			return resource.(*mqlAzureSubscriptionKeyVaultServiceCertificatePolicy), nil
		}
		// Return the actual error for non-404 cases
		return nil, err
	}

	// Extract X.509 properties
	subject := ""
	validityInMonths := int64(0)
	keyUsage := []any{}
	ekus := []any{}

	if policyResp.X509CertificateProperties != nil {
		if policyResp.X509CertificateProperties.Subject != nil {
			subject = *policyResp.X509CertificateProperties.Subject
		}
		if policyResp.X509CertificateProperties.ValidityInMonths != nil {
			validityInMonths = int64(*policyResp.X509CertificateProperties.ValidityInMonths)
		}
		if policyResp.X509CertificateProperties.KeyUsage != nil {
			for _, ku := range policyResp.X509CertificateProperties.KeyUsage {
				if ku != nil {
					keyUsage = append(keyUsage, string(*ku))
				}
			}
		}
		if policyResp.X509CertificateProperties.EnhancedKeyUsage != nil {
			for _, eku := range policyResp.X509CertificateProperties.EnhancedKeyUsage {
				if eku != nil {
					ekus = append(ekus, *eku)
				}
			}
		}
	}

	// Create X.509 properties resource
	x509Props, err := CreateResource(a.MqlRuntime,
		"azure.subscription.keyVaultService.certificate.policy.x509CertificateProperties",
		map[string]*llx.RawData{
			"__id":             llx.StringData(id + "/policy/x509CertificateProperties"),
			"subject":          llx.StringData(subject),
			"validityInMonths": llx.IntData(validityInMonths),
			"keyUsage":         llx.ArrayData(keyUsage, types.String),
			"ekus":             llx.ArrayData(ekus, types.String),
		},
	)
	if err != nil {
		return nil, err
	}

	// Extract key properties
	curve := ""
	exportable := false
	keySize := int64(0)
	keyType := ""
	reuseKey := false

	if policyResp.KeyProperties != nil {
		if policyResp.KeyProperties.Curve != nil {
			curve = string(*policyResp.KeyProperties.Curve)
		}
		if policyResp.KeyProperties.Exportable != nil {
			exportable = *policyResp.KeyProperties.Exportable
		}
		if policyResp.KeyProperties.KeySize != nil {
			keySize = int64(*policyResp.KeyProperties.KeySize)
		}
		if policyResp.KeyProperties.KeyType != nil {
			keyType = string(*policyResp.KeyProperties.KeyType)
		}
		if policyResp.KeyProperties.ReuseKey != nil {
			reuseKey = *policyResp.KeyProperties.ReuseKey
		}
	}

	// Create key properties resource
	keyProps, err := CreateResource(a.MqlRuntime,
		"azure.subscription.keyVaultService.certificate.policy.keyProperties",
		map[string]*llx.RawData{
			"__id":       llx.StringData(id + "/policy/keyProperties"),
			"curve":      llx.StringData(curve),
			"exportable": llx.BoolData(exportable),
			"keySize":    llx.IntData(keySize),
			"keyType":    llx.StringData(keyType),
			"reuseKey":   llx.BoolData(reuseKey),
		},
	)
	if err != nil {
		return nil, err
	}

	// Extract issuer parameters
	certificateTransparency := false
	certificateType := ""
	issuerName := ""

	if policyResp.IssuerParameters != nil {
		if policyResp.IssuerParameters.CertificateTransparency != nil {
			certificateTransparency = *policyResp.IssuerParameters.CertificateTransparency
		}
		if policyResp.IssuerParameters.CertificateType != nil {
			certificateType = *policyResp.IssuerParameters.CertificateType
		}
		if policyResp.IssuerParameters.Name != nil {
			issuerName = *policyResp.IssuerParameters.Name
		}
	}

	// Create issuer parameters resource
	issuerParams, err := CreateResource(a.MqlRuntime,
		"azure.subscription.keyVaultService.certificate.policy.issuerParameters",
		map[string]*llx.RawData{
			"__id":                    llx.StringData(id + "/policy/issuerParameters"),
			"certificateTransparency": llx.BoolData(certificateTransparency),
			"certificateType":         llx.StringData(certificateType),
			"name":                    llx.StringData(issuerName),
		},
	)
	if err != nil {
		return nil, err
	}

	resource, err := CreateResource(a.MqlRuntime,
		"azure.subscription.keyVaultService.certificate.policy",
		map[string]*llx.RawData{
			"__id":                      llx.StringData(id + "/policy"),
			"x509CertificateProperties": llx.ResourceData(x509Props, "azure.subscription.keyVaultService.certificate.policy.x509CertificateProperties"),
			"keyProperties":             llx.ResourceData(keyProps, "azure.subscription.keyVaultService.certificate.policy.keyProperties"),
			"issuerParameters":          llx.ResourceData(issuerParams, "azure.subscription.keyVaultService.certificate.policy.issuerParameters"),
		},
	)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAzureSubscriptionKeyVaultServiceCertificatePolicy), nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceCertificatePolicy) x509CertificateProperties() (*mqlAzureSubscriptionKeyVaultServiceCertificatePolicyX509CertificateProperties, error) {
	if !a.X509CertificateProperties.IsSet() {
		return nil, nil
	}
	if a.X509CertificateProperties.Error != nil {
		return nil, a.X509CertificateProperties.Error
	}
	return a.X509CertificateProperties.Data, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceCertificatePolicy) keyProperties() (*mqlAzureSubscriptionKeyVaultServiceCertificatePolicyKeyProperties, error) {
	if !a.KeyProperties.IsSet() {
		return nil, nil
	}
	if a.KeyProperties.Error != nil {
		return nil, a.KeyProperties.Error
	}
	return a.KeyProperties.Data, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceCertificatePolicy) issuerParameters() (*mqlAzureSubscriptionKeyVaultServiceCertificatePolicyIssuerParameters, error) {
	if !a.IssuerParameters.IsSet() {
		return nil, nil
	}
	if a.IssuerParameters.Error != nil {
		return nil, a.IssuerParameters.Error
	}
	return a.IssuerParameters.Data, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceCertificatePolicy) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceCertificatePolicyX509CertificateProperties) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceCertificatePolicyKeyProperties) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceCertificatePolicyIssuerParameters) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceSecret) secretName() (string, error) {
	id := a.Id.Data
	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return "", err
	}

	return kvid.Name, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceSecret) version() (string, error) {
	id := a.Id.Data
	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return "", err
	}

	return kvid.Version, nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceSecret) versions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	id := a.Id.Data
	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}

	if len(kvid.Version) > 0 {
		return nil, errors.New("cannot fetch versions for an already versioned azure secret")
	}
	if kvid.Type != "secrets" {
		return nil, errors.New("only certificate ids are supported")
	}

	vaultUrl := kvid.BaseUrl
	name := kvid.Name

	ctx := context.Background()
	client, err := azsecrets.NewClient(vaultUrl, conn.Token(), &azsecrets.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListSecretPropertiesVersionsPager(name, &azsecrets.ListSecretPropertiesVersionsOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.keyVaultService.secret",
				map[string]*llx.RawData{
					"id":          llx.StringDataPtr((*string)(entry.ID)),
					"tags":        llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"contentType": llx.StringDataPtr(entry.ContentType),
					"managed":     llx.BoolDataPtr(entry.Managed),
					"enabled":     llx.BoolDataPtr(entry.Attributes.Enabled),
					"created":     llx.TimeDataPtr(entry.Attributes.Created),
					"updated":     llx.TimeDataPtr(entry.Attributes.Updated),
					"expires":     llx.TimeDataPtr(entry.Attributes.Expires),
					"notBefore":   llx.TimeDataPtr(entry.Attributes.NotBefore),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func initAzureSubscriptionKeyVaultServiceVault(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure key vault")
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.keyVaultService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	kv := res.(*mqlAzureSubscriptionKeyVaultService)
	vaults := kv.GetVaults()
	if vaults.Error != nil {
		return nil, nil, vaults.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range vaults.Data {
		vault := entry.(*mqlAzureSubscriptionKeyVaultServiceVault)
		if vault.Id.Data == id {
			return args, vault, nil
		}
	}

	return nil, nil, errors.New("azure key vault does not exist")
}

func (a *mqlAzureSubscriptionKeyVaultServiceKeyRotationPolicyObject) id() (string, error) {
	return a.__id, nil
}
