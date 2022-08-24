package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/vault"
)

func TestVaultConfiguration(t *testing.T) {
	vCfgs := ClientVaultConfig{}

	vCfgs.Set("vault1cfg-key", vault.VaultConfiguration{
		Name: "vault1cfg-name",
	})

	cfg, err := vCfgs.Get("vault1cfg-key")
	require.NoError(t, err)
	assert.Equal(t, "vault1cfg-name", cfg.Name)

	vCfgs.Set("vault1cfg-key", vault.VaultConfiguration{
		Name: "vault1cfg-name2",
	})

	cfg, err = vCfgs.Get("vault1cfg-key")
	require.NoError(t, err)
	assert.Equal(t, "vault1cfg-name2", cfg.Name)

	s := &vault.Secret{
		Key:  "test",
		Data: vCfgs.SecretData(),
	}

	vCfgs2, err := NewClientVaultConfig(s)
	require.NoError(t, err)

	cfg, err = vCfgs2.Get("vault1cfg-key")
	require.NoError(t, err)
	assert.Equal(t, "vault1cfg-name2", cfg.Name)
}
