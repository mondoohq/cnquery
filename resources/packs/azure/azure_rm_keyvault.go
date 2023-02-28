package azure

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azcertificates"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"

	keyvault "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	azure "go.mondoo.com/cnquery/motor/providers/microsoft/azure"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzureSubscriptionKeyvaultService) init(args *resources.Args) (*resources.Args, AzureSubscriptionKeyvaultService, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	(*args)["subscriptionId"] = at.SubscriptionID()

	return args, nil, nil
}

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
// type AzureStorageAccountProperties keyvault_vault.KeyPermissions
// NOTE: the resourcemanager keyvault sdk lacks some functionality/fields for secrets, keys, certs.
// NOTE: instead we use the keyvault/az(certificates/keys/secrets) modules even though they are still in beta.
// NOTE: lets track https://github.com/Azure/azure-sdk-for-go/issues/19412 and see if there's any guidance there once its solved
func (a *mqlAzureSubscriptionKeyvaultService) id() (string, error) {
	subId, err := a.SubscriptionId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/subscriptions/%s/keyVaultService", subId), nil
}

func (a *mqlAzureSubscriptionKeyvaultService) GetVaults() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := keyvault.NewVaultsClient(at.SubscriptionID(), token, &arm.ClientOptions{})
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
			mqlAzure, err := a.MotorRuntime.CreateResource("azure.subscription.keyvaultService.vault",
				"id", core.ToString(entry.ID),
				// TODO: temporary
				"vaultName", core.ToString(entry.Name),
				"location", core.ToString(entry.Location),
				"type", core.ToString(entry.Type),
				"tags", azureTagsToInterface(entry.Tags),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceVault) init(args *resources.Args) (*resources.Args, AzureSubscriptionKeyvaultServiceVault, error) {
	if len(*args) > 1 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(a.MqlResource().MotorRuntime); ids != nil {
			(*args)["id"] = ids.id
		}
	}

	if (*args)["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure keyvault vault")
	}

	obj, err := a.MotorRuntime.CreateResource("azure.subscription.keyvaultService")
	if err != nil {
		return nil, nil, err
	}
	keyvaultSvc := obj.(*mqlAzureSubscriptionKeyvaultService)

	rawResources, err := keyvaultSvc.Vaults()
	if err != nil {
		return nil, nil, err
	}

	id := (*args)["id"].(string)
	for i := range rawResources {
		instance := rawResources[i].(AzureSubscriptionKeyvaultServiceVault)
		instanceId, err := instance.Id()
		if err != nil {
			return nil, nil, errors.New("azure keyvault vault does not exist")
		}
		if instanceId == id {
			return args, instance, nil
		}
	}
	return nil, nil, errors.New("azure keyvault vault does not exist")
}

func (a *mqlAzureSubscriptionKeyvaultServiceVault) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscriptionKeyvaultServiceVault) GetVaultUri() (string, error) {
	name, err := a.VaultName()
	if err != nil {
		return "", err
	}
	KVUri := "https://" + name + ".vault.azure.net"
	return KVUri, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceVault) GetKeys() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	vaultUri, err := a.GetVaultUri()
	if err != nil {
		return nil, err
	}
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := azkeys.NewClient(vaultUri, token, &azkeys.ClientOptions{})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	pager := client.NewListKeysPager(&azkeys.ListKeysOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, entry := range page.Value {
			mqlAzure, err := a.MotorRuntime.CreateResource("azure.subscription.keyvaultService.key",
				"kid", core.ToString((*string)(entry.KID)),
				"managed", core.ToBool(entry.Attributes.Enabled),
				"tags", azureTagsToInterface(entry.Tags),
				"enabled", core.ToBool(entry.Attributes.Enabled),
				"notBefore", entry.Attributes.NotBefore,
				// TODO: handle case where we need to test for a time that is not set
				"expires", entry.Attributes.Expires,
				"created", entry.Attributes.Created,
				"updated", entry.Attributes.Updated,
				"recoveryLevel", core.ToString((*string)(entry.Attributes.RecoveryLevel)),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceVault) GetCertificates() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	vaultUri, err := a.GetVaultUri()
	if err != nil {
		return nil, err
	}

	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := azcertificates.NewClient(vaultUri, token, &azcertificates.ClientOptions{})
	pager := client.NewListCertificatesPager(&azcertificates.ListCertificatesOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzure, err := a.MotorRuntime.CreateResource("azure.subscription.keyvaultService.certificate",
				"id", core.ToString((*string)(entry.ID)),
				"tags", azureTagsToInterface(entry.Tags),
				"x5t", hex.EncodeToString(entry.X509Thumbprint),
				"enabled", core.ToBool(entry.Attributes.Enabled),
				"notBefore", entry.Attributes.NotBefore,
				"expires", entry.Attributes.Expires,
				"created", entry.Attributes.Created,
				"updated", entry.Attributes.Updated,
				"recoveryLevel", core.ToString((*string)(entry.Attributes.RecoveryLevel)),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}

	}
	return res, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceVault) GetSecrets() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	vaultUrl, err := a.VaultUri()
	if err != nil {
		return nil, err
	}

	client, err := azsecrets.NewClient(vaultUrl, token, &azsecrets.ClientOptions{})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	pager := client.NewListSecretsPager(&azsecrets.ListSecretsOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzure, err := a.MotorRuntime.CreateResource("azure.subscription.keyvaultService.secret",
				"id", core.ToString((*string)(entry.ID)),
				"tags", azureTagsToInterface(entry.Tags),
				"contentType", core.ToString(entry.ContentType),
				"managed", core.ToBool(entry.Managed),
				"enabled", core.ToBool(entry.Attributes.Enabled),
				"notBefore", entry.Attributes.NotBefore,
				"expires", entry.Attributes.Expires,
				"created", entry.Attributes.Created,
				"updated", entry.Attributes.Updated,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceVault) GetProperties() (map[string]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := azure.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	vaultName, err := resourceID.Component("vaults")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := keyvault.NewVaultsClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	vault, err := client.Get(ctx, resourceID.ResourceGroup, vaultName, &keyvault.VaultsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(vault.Properties)
}

func (a *mqlAzureSubscriptionKeyvaultServiceVault) GetDiagnosticSettings() ([]interface{}, error) {
	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	// NOTE diagnostics are fetched in the init of azure.monitor.diagnosticsettings
	return diagnosticsSettings(a.MotorRuntime, id)
}

func (a *mqlAzureSubscriptionKeyvaultServiceKey) id() (string, error) {
	return a.Kid()
}

func (a *mqlAzureSubscriptionKeyvaultServiceKey) GetKeyName() (interface{}, error) {
	// parse id "https://superdupervault.vault.azure.net/keys/sqltestkey"
	id, err := a.Kid()
	if err != nil {
		return nil, err
	}

	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}

	return kvid.Name, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceKey) GetVersion() (interface{}, error) {
	id, err := a.Kid()
	if err != nil {
		return nil, err
	}

	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}

	return kvid.Version, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceKey) GetVersions() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	kid, err := a.Kid()
	if err != nil {
		return nil, err
	}
	kvid, err := parseKeyVaultId(kid)
	if err != nil {
		return nil, err
	}
	if len(kvid.Version) > 0 {
		return nil, errors.New("versions is not supported for azure key version")
	}
	if kvid.Type != "keys" {
		return nil, errors.New("only key ids are supported")
	}

	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := azkeys.NewClient(kvid.BaseUrl, token, &azkeys.ClientOptions{})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	pager := client.NewListKeyVersionsPager(kvid.Name, &azkeys.ListKeyVersionsOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzure, err := a.MotorRuntime.CreateResource("azure.subscription.keyvaultService.key",
				"kid", core.ToString((*string)(entry.KID)),
				"managed", core.ToBool(entry.Attributes.Enabled),
				"tags", azureTagsToInterface(entry.Tags),
				"enabled", core.ToBool(entry.Attributes.Enabled),
				"notBefore", entry.Attributes.NotBefore,
				// TODO: handle case where we need to test for a time that is not set
				"expires", entry.Attributes.Expires,
				"created", entry.Attributes.Created,
				"updated", entry.Attributes.Updated,
				"recoveryLevel", core.ToString((*string)(entry.Attributes.RecoveryLevel)),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceCertificate) id() (string, error) {
	return a.Id()
}

// TODO: switch to name once the issue is solved in MQL
func (a *mqlAzureSubscriptionKeyvaultServiceCertificate) GetCertName() (interface{}, error) {
	// parse id "https://superdupervault.vault.azure.net/certificates/testcertificate"
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}
	return kvid.Name, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceCertificate) GetVersion() (interface{}, error) {
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}

	return kvid.Version, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceCertificate) GetX509() (interface{}, error) {
	return nil, errors.New("not implemented")
}

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

func (a *mqlAzureSubscriptionKeyvaultServiceCertificate) GetVersions() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}

	if len(kvid.Version) > 0 {
		return nil, errors.New("versions is not supported for azure certificate version")
	}

	if kvid.Type != "certificates" {
		return nil, errors.New("only certificate ids are supported")
	}

	vaultUrl := kvid.BaseUrl
	name := kvid.Name

	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client := azcertificates.NewClient(vaultUrl, token, &azcertificates.ClientOptions{})

	ctx := context.Background()
	pager := client.NewListCertificateVersionsPager(name, &azcertificates.ListCertificateVersionsOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzure, err := a.MotorRuntime.CreateResource("azure.subscription.keyvaultService.certificate",
				"id", core.ToString((*string)(entry.ID)),
				"tags", azureTagsToInterface(entry.Tags),
				"x5t", hex.EncodeToString(entry.X509Thumbprint),
				"enabled", core.ToBool(entry.Attributes.Enabled),
				"notBefore", entry.Attributes.NotBefore,
				"expires", entry.Attributes.Expires,
				"created", entry.Attributes.Created,
				"updated", entry.Attributes.Updated,
				"recoveryLevel", core.ToString((*string)(entry.Attributes.RecoveryLevel)),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceSecret) id() (string, error) {
	return a.Id()
}

// TODO: switch to name once the issue is solved in MQL
func (a *mqlAzureSubscriptionKeyvaultServiceSecret) GetSecretName() (interface{}, error) {
	// parse id "https://superdupervault.vault.azure.net/certificates/testcertificate"
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}

	return kvid.Name, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceSecret) GetVersion() (interface{}, error) {
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}

	return kvid.Version, nil
}

func (a *mqlAzureSubscriptionKeyvaultServiceSecret) GetVersions() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	kvid, err := parseKeyVaultId(id)
	if err != nil {
		return nil, err
	}

	if len(kvid.Version) > 0 {
		return nil, errors.New("versions is not supported for azure secret version")
	}

	if kvid.Type != "secrets" {
		return nil, errors.New("only secret ids are supported")
	}

	vaultUrl := kvid.BaseUrl
	name := kvid.Name

	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := azsecrets.NewClient(vaultUrl, token, &azsecrets.ClientOptions{})
	if err != nil {
		return nil, err
	}

	pager := client.NewListSecretVersionsPager(name, &azsecrets.ListSecretVersionsOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzure, err := a.MotorRuntime.CreateResource("azure.subscription.keyvaultService.secret",
				"id", core.ToString((*string)(entry.ID)),
				"tags", azureTagsToInterface(entry.Tags),
				"contentType", core.ToString(entry.ContentType),
				"managed", core.ToBool(entry.Managed),
				"enabled", core.ToBool(entry.Attributes.Enabled),
				"notBefore", entry.Attributes.NotBefore,
				"expires", entry.Attributes.Expires,
				"created", entry.Attributes.Created,
				"updated", entry.Attributes.Updated,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}
