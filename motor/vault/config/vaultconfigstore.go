package config

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"errors"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/motor/vault/awsparameterstore"
	"go.mondoo.com/cnquery/motor/vault/awssecretsmanager"
	"go.mondoo.com/cnquery/motor/vault/gcpberglas"
	"go.mondoo.com/cnquery/motor/vault/gcpsecretmanager"
	"go.mondoo.com/cnquery/motor/vault/hashivault"
	"go.mondoo.com/cnquery/motor/vault/inmemory"
	"go.mondoo.com/cnquery/motor/vault/keyring"
)

const (
	VaultConfigStoreName = "mondoo-cli-keyring"
	VaultConfigStoreKey  = "user-vaults"
)

func New(vCfg *vault.VaultConfiguration) (vault.Vault, error) {
	if vCfg == nil {
		return nil, errors.New("vault configuration cannot be empty")
	}
	log.Debug().Str("vault-name", vCfg.Name).Str("vault-type", vCfg.Type.String()).Msg("initialize new vault")
	var v vault.Vault
	switch vCfg.Type {
	case vault.VaultType_Memory:
		v = inmemory.New()
	case vault.VaultType_HashiCorp:
		serverUrl := vCfg.Options["url"]
		token := vCfg.Options["token"]
		v = hashivault.New(serverUrl, token)
	case vault.VaultType_EncryptedFile:
		path := vCfg.Options["path"]
		keyRingName := vCfg.Options["name"]
		password := vCfg.Options["password"]
		v = keyring.NewEncryptedFile(path, keyRingName, password)
	case vault.VaultType_KeyRing:
		keyRingName := vCfg.Options["name"]
		v = keyring.New(keyRingName)
	case vault.VaultType_LinuxKernelKeyring:
		keyRingName := vCfg.Options["name"]
		v = keyring.NewLinuxKernelKeyring(keyRingName)
	case vault.VaultType_GCPSecretsManager:
		projectID := vCfg.Options["project-id"]
		v = gcpsecretmanager.New(projectID)
	case vault.VaultType_AWSSecretsManager:
		// TODO: do we really want to load it from the env?
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, errors.Join(err, errors.New("cannot not determine aws environment"))
		}
		v = awssecretsmanager.New(cfg)
	case vault.VaultType_AWSParameterStore:
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, errors.Join(err, errors.New("cannot not determine aws environment"))
		}
		v = awsparameterstore.New(cfg)
	case vault.VaultType_GCPBerglas:
		projectID := vCfg.Options["project-id"]
		kmsKeyID := vCfg.Options["kms-key-id"]
		bucketName := vCfg.Options["bucket-name"]
		opts := []gcpberglas.Option{}
		if kmsKeyID != "" {
			opts = append(opts, gcpberglas.WithKmsKey(kmsKeyID))
		}
		if bucketName != "" {
			opts = append(opts, gcpberglas.WithBucket(bucketName))
		}
		v = gcpberglas.New(projectID, opts...)

	default:
		return nil, errors.New(fmt.Sprintf("could not connect to vault: %s (%s)", vCfg.Name, vCfg.Type.String()))
	}
	return v, nil
}

// GetInternalVault returns the local store that is used in cnquery to store
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

	log.Debug().Str("vault-name", vCfg.Name).Str("vault-type", vCfg.Type.String()).Msg("found vault config")
	return New(&vCfg)
}

// ClientVaultConfig is the structured type where we store the client configuration for
// all user configured vaults. We use it to ensure the configuration is stored in structured
// format
type ClientVaultConfig map[string]vault.VaultConfiguration

func NewClientVaultConfig(secret *vault.Secret) (ClientVaultConfig, error) {
	var vCfg ClientVaultConfig
	err := json.Unmarshal(secret.Data, &vCfg)
	if err != nil {
		return nil, errors.Join(err, errors.New("corrupt vault configuration"))
	}
	return vCfg, nil
}

func (avc ClientVaultConfig) Delete(key string) {
	delete(avc, key)
}

func (avc ClientVaultConfig) Set(key string, cfg vault.VaultConfiguration) {
	avc[key] = cfg
}

func (avc ClientVaultConfig) Get(key string) (vault.VaultConfiguration, error) {
	vCfg, ok := avc[key]
	if !ok {
		return vault.VaultConfiguration{}, errors.New("vault not found")
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
