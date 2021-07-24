package inventory

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/inventory/v1"
	"go.mondoo.io/mondoo/motor/transports"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestInventoryLoader(t *testing.T) {
	inventory, err := v1.InventoryFromFile("./v1/testdata/inventory.yml")
	require.NoError(t, err)

	im, err := New(WithInventory(inventory))
	require.NoError(t, err)

	// gather all assets and check their secrets
	assetList := im.GetAssets()
	require.NoError(t, err)

	for i := range assetList {
		a := assetList[i]
		for j := range a.Connections {
			conn := a.Connections[j]
			for k := range conn.Credentials {
				cred := conn.Credentials[k]
				_, err := im.GetCredential(cred.SecretId)
				require.NoError(t, err)
			}
		}
	}
}

func TestAssetLoader(t *testing.T) {
	_, err := New(WithAssets([]*asset.Asset{
		{
			Name: "test asset",
		},
	}))
	require.NoError(t, err)
}

func TestSecretCredentialConversion(t *testing.T) {
	cred := &transports.Credential{
		Type:     transports.CredentialType_password,
		User:     "username",
		Password: "pass1",
	}

	secret, err := NewSecret(cred)
	require.NoError(t, err)

	cred2, err := NewCredential(secret)
	require.NoError(t, err)

	if d := cmp.Diff(cred, cred2, protocmp.Transform()); d != "" {
		t.Error("credentials are different", d)
	}
}
