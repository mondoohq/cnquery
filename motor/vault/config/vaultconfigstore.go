package config

import (
	"context"
	"encoding/json"
	"runtime"

	"github.com/rs/zerolog/log"

	"go.mondoo.io/mondoo/motor/vault/keyring"

	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/motor/vault"
)

const (
	VaultConfigStoreName = "mondoo-cli-keyring"
	VaultConfigStoreKey  = "user-vaults"
)

// GetInternalVault returns the local store that is used in Mondoo client to store
// Vault configurations eg. Hashicorp Vault access data
func GetInternalVault() vault.Vault {
	// on linux we are going to use kernel key management
	if runtime.GOOS == "linux" {
		log.Debug().Msg("use linux kernel key management to manage vaults")
		return keyring.NewLinuxKernelKeyring(VaultConfigStoreName)
	}

	return keyring.New(VaultConfigStoreName)
}

// GetConfiguredVault returns a vault instance based on the configured user vaults.
// It looks up in the internal vault and searches for a configuration for the vaultName
func GetConfiguredVault(vaultName string) (vault.Vault, error) {
	v := GetInternalVault()

	ctx := context.Background()
	secret, err := v.Get(ctx, &vault.SecretID{
		Key: VaultConfigStoreKey,
	})
	if err != nil {
		return nil, err
	}

	cfgs, err := NewClientVaultConfig(secret)
	if err != nil {
		return nil, err
	}

	// search for the specified vault
	vCfg, err := cfgs.Get(vaultName)
	if err != nil {
		return nil, err
	}

	log.Debug().Str("vault-name", vCfg.Name).Str("vault-type", vCfg.VaultType).Msg("found vault config")
	return New(vCfg)
}

// ClientVaultConfig is the structured type where we store the client configuration for
// all user configured vaults. We use it to ensure the configuration is stored in structured
// format
type ClientVaultConfig map[string]VaultConfiguration

func NewClientVaultConfig(secret *vault.Secret) (ClientVaultConfig, error) {
	var vCfg ClientVaultConfig
	err := json.Unmarshal(secret.Data, &vCfg)
	if err != nil {
		return nil, errors.Wrap(err, "corrupt vault configuration")
	}
	return vCfg, nil
}

func (avc ClientVaultConfig) Delete(key string) {
	delete(avc, key)
}

func (avc ClientVaultConfig) Set(key string, cfg VaultConfiguration) {
	avc[key] = cfg
}

func (avc ClientVaultConfig) Get(key string) (VaultConfiguration, error) {
	vCfg, ok := avc[key]
	if !ok {
		return VaultConfiguration{}, errors.New("vault not found")
	}
	return vCfg, nil
}

// SecretData returns the marshaled data, it is compatible with New()
// In case the data structure cannot be marshalled, the function will panic
func (avc ClientVaultConfig) SecretData() []byte {
	data, err := json.Marshal(avc)
	if err != nil {
		panic(err)
	}
	return data
}
