// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
)

func TestInventoryParser(t *testing.T) {
	inventory, err := InventoryFromFile("./testdata/inventory.yaml")
	require.NoError(t, err)
	require.NotNil(t, inventory)

	assert.Equal(t, "mondoo-inventory", inventory.Metadata.Name)
	assert.Equal(t, "production", inventory.Metadata.Labels["environment"])
	assert.Equal(t, "{ id: 'secret-1' }", inventory.Spec.CredentialQuery)
}

func TestPreprocess(t *testing.T) {
	t.Run("preprocess empty inventory", func(t *testing.T) {
		v1inventory := &Inventory{}
		err := v1inventory.PreProcess()
		require.NoError(t, err)
	})

	t.Run("normal inventory", func(t *testing.T) {
		inventory, err := InventoryFromFile("./testdata/inventory.yaml")
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
	})

	t.Run("idempotent preprocess", func(t *testing.T) {
		v1inventory, err := InventoryFromFile("./testdata/k8s_mount.yaml")
		require.NoError(t, err)

		err = v1inventory.PreProcess()
		require.NoError(t, err)

		err = v1inventory.PreProcess()
		require.NoError(t, err)
	})

	t.Run("preprocess private key", func(t *testing.T) {
		v1inventory := &Inventory{
			Spec: &InventorySpec{
				Assets: []*Asset{
					{
						Name: "test",
						Connections: []*Config{
							{
								Type: "ssh",
								Credentials: []*vault.Credential{
									{
										PrivateKey: "./testdata/private_key_01",
									},
								},
							},
						},
					},
				},
			},
		}
		err := v1inventory.PreProcess()
		require.NoError(t, err)
		secretid := v1inventory.Spec.Assets[0].Connections[0].Credentials[0].SecretId
		assert.Equal(t, vault.CredentialType_private_key, v1inventory.Spec.Credentials[secretid].Type)
	})

	t.Run("preprocess pkcs12 credential with loading from file", func(t *testing.T) {
		v1inventory := &Inventory{
			Spec: &InventorySpec{
				Assets: []*Asset{
					{
						Name: "test",
						Connections: []*Config{
							{
								Type: "ms365",
								Credentials: []*vault.Credential{
									{
										Type:           vault.CredentialType_pkcs12,
										PrivateKeyPath: "./testdata/private_key_01",
									},
								},
							},
						},
					},
				},
			},
		}
		err := v1inventory.PreProcess()
		require.NoError(t, err)
		secretid := v1inventory.Spec.Assets[0].Connections[0].Credentials[0].SecretId
		assert.Equal(t, vault.CredentialType_pkcs12, v1inventory.Spec.Credentials[secretid].Type)
	})

	t.Run("preprocess pkcs12 credential with loading from file", func(t *testing.T) {
		v1inventory := &Inventory{
			Spec: &InventorySpec{
				Assets: []*Asset{
					{
						Name: "test",
						Connections: []*Config{
							{
								Type: "ms365",
								Credentials: []*vault.Credential{
									{
										Type:       vault.CredentialType_pkcs12,
										PrivateKey: "secretdata",
									},
								},
							},
						},
					},
				},
			},
		}
		err := v1inventory.PreProcess()
		require.NoError(t, err)
		secretid := v1inventory.Spec.Assets[0].Connections[0].Credentials[0].SecretId
		assert.Equal(t, vault.CredentialType_pkcs12, v1inventory.Spec.Credentials[secretid].Type)
	})
}

func TestParseGCPInventory(t *testing.T) {
	inventory, err := InventoryFromFile("./testdata/gcp_inventory.yaml")
	require.NoError(t, err)

	// extract credentials into credential section
	err = inventory.PreProcess()
	require.NoError(t, err)

	assert.Equal(t, "gcp", inventory.Spec.Assets[0].Connections[0].Type)
	// ensure that all assets have a valid secret reference
	err = inventory.Validate()
	require.NoError(t, err)
}

func TestParseVsphereInventory(t *testing.T) {
	inventory, err := InventoryFromFile("./testdata/vsphere_inventory.yaml")
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

func TestParseSshInventory(t *testing.T) {
	inventory, err := InventoryFromFile("./testdata/ssh_inventory.yaml")
	require.NoError(t, err)

	// extract credentials into credential section
	err = inventory.PreProcess()
	require.NoError(t, err)

	// ensure that all assets have a valid secret reference
	err = inventory.Validate()
	require.NoError(t, err)

	a := findAsset(inventory.Spec.Assets, "linux-with-password")
	require.NotNil(t, a)

	assert.Equal(t, vault.CredentialType_password, inventory.Spec.Credentials[a.Connections[0].Credentials[0].SecretId].Type)

	a = findAsset(inventory.Spec.Assets, "linux-ssh-agent")
	require.NotNil(t, a)
	assert.Equal(t, vault.CredentialType_ssh_agent, inventory.Spec.Credentials[a.Connections[0].Credentials[0].SecretId].Type)

	a = findAsset(inventory.Spec.Assets, "linux-identity-key")
	require.NotNil(t, a)
	assert.Equal(t, vault.CredentialType_private_key, inventory.Spec.Credentials[a.Connections[0].Credentials[0].SecretId].Type)
}

func TestParseVaultInventory(t *testing.T) {
	inventory, err := InventoryFromFile("./testdata/vault_inventory.yaml")
	require.NoError(t, err)

	// extract credentials into credential section
	err = inventory.PreProcess()
	require.NoError(t, err)

	// ensure that all assets have a valid secret reference
	err = inventory.Validate()
	require.NoError(t, err)
}

func TestNilPointer(t *testing.T) {
	inventory, err := InventoryFromFile("./testdata/no_metadata_inventory.yaml")
	require.NoError(t, err)

	assert.NotNil(t, inventory.Metadata)
	assert.NotNil(t, inventory.Metadata.Labels)
}

func TestMarkInsecure(t *testing.T) {
	inventory, err := InventoryFromFile("./testdata/ssh_inventory.yaml")
	require.NoError(t, err)

	// extract credentials into credential section
	err = inventory.PreProcess()
	require.NoError(t, err)

	// check that all assets have no insecure flag set
	for i := range inventory.Spec.Assets {
		a := inventory.Spec.Assets[i]
		for j := range a.Connections {
			assert.False(t, a.Connections[j].Insecure, a.Name)
		}
	}

	inventory.MarkConnectionsInsecure()

	// check that all connections are marked as insecure
	for i := range inventory.Spec.Assets {
		a := inventory.Spec.Assets[i]
		for j := range a.Connections {
			assert.True(t, a.Connections[j].Insecure, a.Name)
		}
	}
}

func findAsset(assets []*Asset, id string) *Asset {
	for i := range assets {
		a := assets[i]
		if a.Id == id {
			return a
		}
	}
	return nil
}
