package inventory

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/asset"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
)

func TestInventoryLoader(t *testing.T) {
	inventory, err := v1.InventoryFromFile("./v1/testdata/inventory.yaml")
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
				_, err := im.GetCredential(cred)
				assert.NoError(t, err, cred.SecretId)
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

func TestAwsInventoryLoader(t *testing.T) {
	inventory, err := v1.InventoryFromFile("./v1/testdata/aws_inventory.yaml")
	require.NoError(t, err)

	os.Setenv("AWS_PROFILE", "mondoo-dev")
	os.Setenv("AWS_REGION", "us-east-1")

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
				resolvedCred, err := im.GetCredential(cred)
				assert.NoError(t, err, cred.SecretId)
				assert.NotNil(t, resolvedCred)
			}
		}
	}
}

func TestVaultInventoryLoader(t *testing.T) {
	// load inventory with vault name, it will throw an error since vault is not configured fully
	inventory, err := v1.InventoryFromFile("./v1/testdata/vault_inventory.yaml")
	require.NoError(t, err)

	_, err = New(WithInventory(inventory))
	require.Error(t, err)
}
