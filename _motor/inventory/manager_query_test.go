package inventory

import (
	"testing"

	"go.mondoo.com/cnquery/motor/vault"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/vault/credentials_resolver"
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

	assetObj := &v1.Asset{
		Name:     "asset-name",
		Platform: &v1.Platform{Name: "ubuntu"},
		Connections: []*v1.Config{
			{Type: "ssh", Insecure: true},
		},
	}

	credential, err := im.QuerySecretId(assetObj)
	require.NoError(t, err)

	assert.Equal(t, vault.CredentialType_password, credential.Type)
	assert.Equal(t, "test-user", credential.User)
	assert.Equal(t, "mockPassword", credential.SecretId)

	// now we try to get the full credential with the secret
	credsResolver := credentials_resolver.New(im.GetVault(), false)
	_, err = credsResolver.GetCredential(credential)
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

	assetObj := &v1.Asset{
		Name:     "asset-name",
		Platform: &v1.Platform{Name: "ubuntu"},
		Connections: []*v1.Config{
			{Type: "ssh", Insecure: true},
		},
	}

	credential, err := im.QuerySecretId(assetObj)
	require.NoError(t, err)

	assert.Equal(t, vault.CredentialType_private_key, credential.Type)
	assert.Equal(t, "some-user", credential.User)
	assert.Equal(t, "mockPKey", credential.SecretId)

	// now we try to get the full credential with the secret
	credsResolver := credentials_resolver.New(im.GetVault(), false)
	_, err = credsResolver.GetCredential(credential)
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

	assetObj := &v1.Asset{
		Name:     "asset-name",
		Platform: &v1.Platform{Name: "ubuntu"},
		Connections: []*v1.Config{
			{Type: "ssh", Insecure: true},
		},
	}

	// NOTE: we get the secret id but the load from the vault will fail
	credential, err := im.QuerySecretId(assetObj)
	assert.NoError(t, err)
	assert.Equal(t, vault.CredentialType_password, credential.Type)
	assert.Equal(t, "some-user", credential.User)
	assert.Equal(t, "bad-id", credential.SecretId)

	// now we try to get the full credential with the secret
	credsResolver := credentials_resolver.New(im.GetVault(), false)
	_, err = credsResolver.GetCredential(credential)
	assert.Error(t, err)
}
