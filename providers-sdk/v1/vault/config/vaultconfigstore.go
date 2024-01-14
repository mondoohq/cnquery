// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault/awsparameterstore"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault/awssecretsmanager"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault/gcpberglas"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault/gcpsecretmanager"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault/hashivault"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault/inmemory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault/keyring"
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
		keyRingName := vCfg.Name
		password := vCfg.Options["password"]
		v = keyring.NewEncryptedFile(path, keyRingName, password)
	case vault.VaultType_KeyRing:
		keyRingName := vCfg.Name
		v = keyring.New(keyRingName)
	case vault.VaultType_LinuxKernelKeyring:
		keyRingName := vCfg.Name
		v = keyring.NewLinuxKernelKeyring(keyRingName)
	case vault.VaultType_GCPSecretsManager:
		projectID := vCfg.Options["project-id"]
		v = gcpsecretmanager.New(projectID)
	case vault.VaultType_AWSSecretsManager:
		// TODO: do we really want to load it from the env?
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "cannot not determine aws environment")
		}
		v = awssecretsmanager.New(cfg)
	case vault.VaultType_AWSParameterStore:
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "cannot not determine aws environment")
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
		return nil, errors.Errorf("could not connect to vault: %s (%s)", vCfg.Name, vCfg.Type.String())
	}
	return v, nil
}

// ClientVaultConfig is the structured type where we store the client configuration for
// all user configured vaults. We use it to ensure the configuration is stored in structured
// format
type ClientVaultConfig map[string]vault.VaultConfiguration

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
