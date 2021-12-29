package config

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/vault"
	"go.mondoo.io/mondoo/motor/vault/awsparameterstore"
	"go.mondoo.io/mondoo/motor/vault/awssecretsmanager"
	"go.mondoo.io/mondoo/motor/vault/gcpsecretmanager"
	"go.mondoo.io/mondoo/motor/vault/hashivault"
	"go.mondoo.io/mondoo/motor/vault/keyring"
	"go.mondoo.io/mondoo/stringx"
)

const (
	Vault_Hashicorp          string = "hashicorp-vault"
	Vault_EncryptedFile      string = "encrypted-file"
	Vault_Keyring            string = "keyring"
	Vault_GCPSecretsManager  string = "gcp-secret-manager"
	Vault_AWSSecretsManager  string = "aws-secrets-manager"
	Vault_AWSParameterStore  string = "aws-parameter-store"
	Vault_LinuxKernelKeyring string = "linux-kernel-keyring"
)

var SupportedVaultTypes = []string{
	Vault_Hashicorp,
	Vault_EncryptedFile,
	Vault_Keyring,
	Vault_GCPSecretsManager,
	Vault_AWSSecretsManager,
	Vault_AWSParameterStore,
	Vault_LinuxKernelKeyring,
}

type VaultConfiguration struct {
	Name      string            `json:"name,omitempty"`
	VaultType string            `json:"type,omitempty"`
	Options   map[string]string `json:"options,omitempty"`
}

func (vc VaultConfiguration) Validate() error {
	if !stringx.Contains(SupportedVaultTypes, vc.VaultType) {
		return errors.Errorf("unsupported vault type: %s, use one of: %s", vc.VaultType, strings.Join(SupportedVaultTypes, ","))
	}
	return nil
}

func New(vCfg VaultConfiguration) (vault.Vault, error) {
	log.Debug().Str("vault-name", vCfg.Name).Str("vault-type", vCfg.VaultType).Msg("initialize new vault")
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
	case Vault_LinuxKernelKeyring:
		keyRingName := vCfg.Options["name"]
		vault = keyring.NewLinuxKernelKeyring(keyRingName)
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
		return nil, errors.Errorf("could not connect to vault: %s (%s)", vCfg.Name, vCfg.VaultType)
	}
	return vault, nil
}
