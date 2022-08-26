package inventory

import (
	"testing"

	"go.mondoo.com/cnquery/motor/vault"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/asset"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	mockvault "go.mondoo.com/cnquery/motor/vault/mock"
)

func TestSecretManagerPassword(t *testing.T) {
	im, err := New(
		WithInventory(&v1.Inventory{
			Spec: &v1.InventorySpec{
				CredentialQuery: "{type: 'password', secret_id: 'mockPassword', user: 'test-user'}",
			},
		}),
		WithVault(mockvault.New()),
	)
	require.NoError(t, err)

	assetObj := &asset.Asset{
		Name:     "asset-name",
		Platform: &platform.Platform{Name: "ubuntu"},
		Connections: []*providers.Config{
			{Backend: providers.ProviderType_SSH, Insecure: true},
		},
	}

	credential, err := im.QuerySecretId(assetObj)
	require.NoError(t, err)

	assert.Equal(t, vault.CredentialType_password, credential.Type)
	assert.Equal(t, "test-user", credential.User)
	assert.Equal(t, "mockPassword", credential.SecretId)

	// now we try to get the full credential with the secret
	_, err = im.GetCredential(credential)
	assert.NoError(t, err)
}

func TestSecretManagerPrivateKey(t *testing.T) {
	im, err := New(
		WithInventory(&v1.Inventory{
			Spec: &v1.InventorySpec{
				CredentialQuery: "{type: 'private_key',  secret_id: 'mockPKey', user: 'some-user'}",
			},
		}),
		WithVault(mockvault.New()),
	)
	require.NoError(t, err)

	assetObj := &asset.Asset{
		Name:     "asset-name",
		Platform: &platform.Platform{Name: "ubuntu"},
		Connections: []*providers.Config{
			{Backend: providers.ProviderType_SSH, Insecure: true},
		},
	}

	credential, err := im.QuerySecretId(assetObj)
	require.NoError(t, err)

	assert.Equal(t, vault.CredentialType_private_key, credential.Type)
	assert.Equal(t, "some-user", credential.User)
	assert.Equal(t, "mockPKey", credential.SecretId)

	// now we try to get the full credential with the secret
	_, err = im.GetCredential(credential)
	assert.NoError(t, err)
}

func TestSecretManagerBadKey(t *testing.T) {
	im, err := New(
		WithInventory(&v1.Inventory{
			Spec: &v1.InventorySpec{
				CredentialQuery: "{type: 'password',  secret_id: 'bad-id', user: 'some-user'}",
			},
		}),
		WithVault(mockvault.New()),
	)
	require.NoError(t, err)

	assetObj := &asset.Asset{
		Name:     "asset-name",
		Platform: &platform.Platform{Name: "ubuntu"},
		Connections: []*providers.Config{
			{Backend: providers.ProviderType_SSH, Insecure: true},
		},
	}

	// NOTE: we get the secret id but the load from the vault will fail
	credential, err := im.QuerySecretId(assetObj)
	assert.NoError(t, err)
	assert.Equal(t, vault.CredentialType_password, credential.Type)
	assert.Equal(t, "some-user", credential.User)
	assert.Equal(t, "bad-id", credential.SecretId)

	// now we try to get the full credential with the secret
	_, err = im.GetCredential(credential)
	assert.Error(t, err)
}
