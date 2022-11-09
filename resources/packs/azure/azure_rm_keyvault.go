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
	"go.mondoo.com/cnquery/resources/packs/core"
)

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
// type AzureStorageAccountProperties keyvault_vault.KeyPermissions
// NOTE: the resourcemanager keyvault sdk lacks some functionality/fields for secrets, keys, certs.
// NOTE: instead we use the keyvault/az(certificates/keys/secrets) modules even though they are still in beta.
// NOTE: lets track https://github.com/Azure/azure-sdk-for-go/issues/19412 and see if there's any guidance there once its solved
func (a *mqlAzurermKeyvault) id() (string, error) {
	return "azure.keyvault", nil
}

func (a *mqlAzurermKeyvault) GetVaults() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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
			mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.keyvault.vault",
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

func (a *mqlAzurermKeyvaultVault) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermKeyvaultVault) GetVaultUri() (string, error) {
	name, err := a.VaultName()
	if err != nil {
		return "", err
	}
	KVUri := "https://" + name + ".vault.azure.net"
	return KVUri, nil
}

func (a *mqlAzurermKeyvaultVault) GetKeys() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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

	client := azkeys.NewClient(vaultUri, token, &azkeys.ClientOptions{})

	ctx := context.Background()
	pager := client.NewListKeysPager(&azkeys.ListKeysOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, entry := range page.Value {
			mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.keyvault.key",
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

func (a *mqlAzurermKeyvaultVault) GetCertificates() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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
			mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.keyvault.certificate",
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

func (a *mqlAzurermKeyvaultVault) GetSecrets() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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

	client := azsecrets.NewClient(vaultUrl, token, &azsecrets.ClientOptions{})
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
			mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.keyvault.secret",
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

func (a *mqlAzurermKeyvaultVault) GetProperties() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
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

func (a *mqlAzurermKeyvaultVault) GetDiagnosticSettings() ([]interface{}, error) {
	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	// NOTE diagnostics are fetched in the init of azurerm.monitor.diagnosticsettings
	return diagnosticsSettings(a.MotorRuntime, id)
}

func (a *mqlAzurermKeyvaultKey) id() (string, error) {
	return a.Kid()
}

func (a *mqlAzurermKeyvaultKey) GetKeyName() (interface{}, error) {
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

func (a *mqlAzurermKeyvaultKey) GetVersion() (interface{}, error) {
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

func (a *mqlAzurermKeyvaultKey) GetVersions() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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

	client := azkeys.NewClient(kvid.BaseUrl, token, &azkeys.ClientOptions{})
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
			mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.keyvault.key",
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

func (a *mqlAzurermKeyvaultCertificate) id() (string, error) {
	return a.Id()
}

// TODO: switch to name once the issue is solved in MQL
func (a *mqlAzurermKeyvaultCertificate) GetCertName() (interface{}, error) {
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

func (a *mqlAzurermKeyvaultCertificate) GetVersion() (interface{}, error) {
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

func (a *mqlAzurermKeyvaultCertificate) GetX509() (interface{}, error) {
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

func (a *mqlAzurermKeyvaultCertificate) GetVersions() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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
			mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.keyvault.certificate",
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

func (a *mqlAzurermKeyvaultSecret) id() (string, error) {
	return a.Id()
}

// TODO: switch to name once the issue is solved in MQL
func (a *mqlAzurermKeyvaultSecret) GetSecretName() (interface{}, error) {
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

func (a *mqlAzurermKeyvaultSecret) GetVersion() (interface{}, error) {
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

func (a *mqlAzurermKeyvaultSecret) GetVersions() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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
	client := azsecrets.NewClient(vaultUrl, token, &azsecrets.ClientOptions{})
	pager := client.NewListSecretVersionsPager(name, &azsecrets.ListSecretVersionsOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.keyvault.secret",
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
