package config

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/motor/vault"
	"go.mondoo.io/mondoo/motor/vault/awsparameterstore"
	"go.mondoo.io/mondoo/motor/vault/awssecretsmanager"
	"go.mondoo.io/mondoo/motor/vault/gcpsecretmanager"
	"go.mondoo.io/mondoo/motor/vault/hashivault"
	"go.mondoo.io/mondoo/motor/vault/keyring"
)

type VaultConfiguration struct {
	Name      string            `json:"name,omitempty"`
	VaultType string            `json:"type,omitempty" `
	Options   map[string]string `json:"options,omitempty" `
}

const (
	Vault_Hashicorp         string = "hashicorp-vault"
	Vault_EncryptedFile     string = "encrypted-file"
	Vault_Keyring           string = "keyring"
	Vault_GCPSecretsManager string = "gcp-secret-manager"
	Vault_AWSSecretsManager string = "aws-secrets-manager"
	Vault_AWSParameterStore string = "aws-parameter-store"
)

func New(vCfg VaultConfiguration) (vault.Vault, error) {
	var vault vault.Vault
	switch vCfg.VaultType {
	case Vault_Hashicorp:
		serverUrl := vCfg.Options["url"]
		token := vCfg.Options["token"]
		vault = hashivault.New(serverUrl, token)
	case Vault_EncryptedFile:
		path := vCfg.Options["path"]
		keyRingName := vCfg.Options["name"]
		password := vCfg.Options["password"]
		vault = keyring.NewEncryptedFile(path, keyRingName, password)
	case Vault_Keyring:
		keyRingName := vCfg.Options["name"]
		vault = keyring.New(keyRingName)
	case Vault_GCPSecretsManager:
		projectID := vCfg.Options["project-id"]
		vault = gcpsecretmanager.New(projectID)
	case Vault_AWSSecretsManager:
		// TODO: do we really want to load it from the env?
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "cannot not determine aws environment")
		}
		vault = awssecretsmanager.New(cfg)
	case Vault_AWSParameterStore:
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "cannot not determine aws environment")
		}
		vault = awsparameterstore.New(cfg)
	default:
		return nil, errors.New("the vault type is unknown: " + vCfg.VaultType)
	}
	return vault, nil
}
