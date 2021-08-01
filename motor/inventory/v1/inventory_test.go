package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestInventoryParser(t *testing.T) {
	inventory, err := InventoryFromFile("./testdata/inventory.yml")
	require.NoError(t, err)
	require.NotNil(t, inventory)

	assert.Equal(t, "mondoo-inventory", inventory.Metadata.Name)
	assert.Equal(t, "production", inventory.Metadata.Labels["environment"])
	assert.Equal(t, "{ id: 'secret-1' }", inventory.Spec.CredentialQuery)
}

func TestPreprocess(t *testing.T) {
	inventory, err := InventoryFromFile("./testdata/inventory.yml")
	require.NoError(t, err)

	// extract credentials into credential section
	err = inventory.PreProcess()
	require.NoError(t, err)

	// ensure that all assets have a valid secret reference
	err = inventory.Validate()
	require.NoError(t, err)

	// activate to debug the pre-process output
	//// write output for debugging, so that we can easily compare the result
	//data, err := inventory.ToYAML()
	//require.NoError(t, err)
	//
	//err = ioutil.WriteFile("./testdata/inventory.parsed.yml", data, 0o700)
	//require.NoError(t, err)
}

func TestParseGCPInventory(t *testing.T) {
	inventory, err := InventoryFromFile("./testdata/gcp_inventory.yml")
	require.NoError(t, err)

	// extract credentials into credential section
	err = inventory.PreProcess()
	require.NoError(t, err)

	// ensure that all assets have a valid secret reference
	err = inventory.Validate()
	require.NoError(t, err)
}

func TestParseVsphereInventory(t *testing.T) {
	inventory, err := InventoryFromFile("./testdata/vsphere_inventory.yml")
	require.NoError(t, err)

	// extract credentials into credential section
	err = inventory.PreProcess()
	require.NoError(t, err)

	// ensure that all assets have a valid secret reference
	err = inventory.Validate()
	require.NoError(t, err)

	// check that the password was pre-processed
	cred := inventory.Spec.Assets[0].Connections[0].Credentials[0]
	assert.Equal(t, "", cred.User)
	assert.Equal(t, "", cred.Password)
	assert.Equal(t, []byte{}, cred.Secret)

	secret := inventory.Spec.Credentials[cred.SecretId]
	assert.Equal(t, "root", secret.User)
	assert.Equal(t, "", secret.Password)
	assert.Equal(t, []byte("password1!"), secret.Secret)
}
