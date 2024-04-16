// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
	"go.mondoo.com/cnquery/v11/types"

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

	conn := runtime.Connection.(*connection.AzureConnection)
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

func (a *mqlAzureSubscriptionKeyVaultService) vaults() ([]interface{}, error) {
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
	res := []interface{}{}

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

func (a *mqlAzureSubscriptionKeyVaultServiceVault) properties() (interface{}, error) {
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
	propsDict := props.Data.(map[string]interface{})
	rbacProp := propsDict["enableRbacAuthorization"]
	if rbacProp == nil {
		return false, errors.New("key vault does not have enableRbacAuthorization property")
	}
	return rbacProp.(bool), nil
}

func (a *mqlAzureSubscriptionKeyVaultServiceVault) keys() ([]interface{}, error) {
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
	res := []interface{}{}
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

func (a *mqlAzureSubscriptionKeyVaultServiceVault) secrets() ([]interface{}, error) {
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
	res := []interface{}{}
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

func (a *mqlAzureSubscriptionKeyVaultServiceVault) certificates() ([]interface{}, error) {
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
	res := []interface{}{}
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

func (a *mqlAzureSubscriptionKeyVaultServiceVault) diagnosticSettings() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	return getDiagnosticSettings(a.Id.Data, a.MqlRuntime, conn)
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

func (a *mqlAzureSubscriptionKeyVaultServiceKey) versions() ([]interface{}, error) {
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
	res := []interface{}{}
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

func (a *mqlAzureSubscriptionKeyVaultServiceCertificate) versions() ([]interface{}, error) {
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
	res := []interface{}{}
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

func (a *mqlAzureSubscriptionKeyVaultServiceSecret) versions() ([]interface{}, error) {
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
	res := []interface{}{}
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

	conn := runtime.Connection.(*connection.AzureConnection)
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
