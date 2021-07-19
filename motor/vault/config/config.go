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

func New(vCfg VaultConfiguration) (vault.Vault, error) {
	var vault vault.Vault
	switch vCfg.VaultType {
	case "hashicorp-vault":
		serverUrl := vCfg.Options["url"]
		token := vCfg.Options["token"]
		vault = hashivault.New(serverUrl, token)
	case "encrypted-file":
		path := vCfg.Options["path"]
		keyRingName := vCfg.Options["name"]
		password := vCfg.Options["password"]
		vault = keyring.NewEncryptedFile(path, keyRingName, password)
	case "keyring":
		keyRingName := vCfg.Options["name"]
		vault = keyring.New(keyRingName)
	case "gcp-secret-manager":
		projectID := vCfg.Options["project-id"]
		vault = gcpsecretmanager.New(projectID)
	case "aws-secrets-manager":
		// TODO: do we really want to load it from the env?
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "cannot not determine aws environment")
		}
		vault = awssecretsmanager.New(cfg)
	case "aws-parameter-store":
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
