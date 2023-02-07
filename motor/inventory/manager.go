package inventory

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory/credentialquery"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/motor/vault/config"
	"go.mondoo.com/cnquery/motor/vault/credentials_resolver"
	"go.mondoo.com/cnquery/motor/vault/inmemory"
	"go.mondoo.com/cnquery/motor/vault/multivault"
)

var _ InventoryManager = (*inventoryManager)(nil)

type InventoryManager interface {
	// GetAssets returns all assets under management
	GetAssets() []*asset.Asset
	// GetRelatedAssets returns a list of assets related to those under management
	GetRelatedAssets() []*asset.Asset
	// Resolve will iterate over all assets and try to discover all nested assets. After this operation
	// GetAssets will return the fully resolved list of assets
	Resolve(ctx context.Context) map[*asset.Asset]error
	// GetCredential returns a full credential including the secret from vault
	GetCredential(*vault.Credential) (*vault.Credential, error)
	// QuerySecretId runs the credential query to determine the secret id for an asset, the resulting credential
	// only returns a secret id
	QuerySecretId(a *asset.Asset) (*vault.Credential, error)
	// GetVault returns the configured Vault
	GetVault() vault.Vault
	GetCredsResolver() vault.Resolver
}

type Option func(*inventoryManager) error

// passes a pre-parsed asset inventory into the Inventory Manager
func WithInventory(inventory *v1.Inventory) Option {
	return func(im *inventoryManager) error {
		logger.DebugDumpJSON("inventory-unresolved", inventory)
		return im.loadInventory(inventory)
	}
}

// passes a list of asset into the Inventory Manager
func WithAssets(assetList []*asset.Asset) Option {
	return func(im *inventoryManager) error {
		return im.loadInventory(v1.New(v1.WithAssets(assetList...)))
	}
}

func WithCredentialQuery(query string) Option {
	return func(im *inventoryManager) error {
		return im.SetCredentialQuery(query)
	}
}

func WithVault(v vault.Vault) Option {
	return func(im *inventoryManager) error {
		im.vault = v
		return nil
	}
}

func WithCachedCredsResolver() Option {
	return func(im *inventoryManager) error {
		im.isCached = true
		return nil
	}
}

func New(opts ...Option) (*inventoryManager, error) {
	im := &inventoryManager{
		assetList: []*asset.Asset{},
	}

	for _, option := range opts {
		if err := option(im); err != nil {
			return nil, err
		}
	}
	im.resetVault()

	return im, nil
}

type inventoryManager struct {
	isCached      bool
	assetList     []*asset.Asset
	relatedAssets []*asset.Asset
	// optional vault set by user
	vault vault.Vault
	// internal vault used to store embedded credentials
	inmemoryVault vault.Vault
	// wrapper vault to access the credentials
	accessVault vault.Vault

	credentialQueryRunner *credentialquery.CredentialQueryRunner
}

// TODO: define what happens when we call load multiple times?
func (im *inventoryManager) loadInventory(inventory *v1.Inventory) error {
	err := inventory.PreProcess()
	if err != nil {
		return err
	}

	// all assets are copied
	im.assetList = append(im.assetList, inventory.Spec.Assets...)

	// palace all credentials in an in-memory vault
	secrets := map[string]*vault.Secret{}
	for i := range inventory.Spec.Credentials {
		cred := inventory.Spec.Credentials[i]

		secret, err := vault.NewSecret(cred, vault.SecretEncoding_encoding_proto)
		if err != nil {
			return err
		}

		secrets[secret.Key] = secret
	}

	if inventory.Spec.CredentialQuery != "" {
		err = im.SetCredentialQuery(inventory.Spec.CredentialQuery)
		if err != nil {
			return err
		}
	}

	// in-memory vault is used as fall-back store embedded credentials
	im.inmemoryVault = inmemory.New(inmemory.WithSecretMap(secrets))
	if inventory.Spec.Vault != nil {
		var v vault.Vault
		// when the type is not provided but a name was given, then look up in our internal vault configuration
		if inventory.Spec.Vault.Name != "" && inventory.Spec.Vault.Type == "" {
			v, err = config.GetConfiguredVault(inventory.Spec.Vault.Name)
			if err != nil {
				return err
			}
		} else {
			t, err := vault.NewVaultType(inventory.Spec.Vault.Type)
			if err != nil {
				return err
			}

			// instantiate with full vault config
			v, err = config.New(&vault.VaultConfiguration{
				Name:    inventory.Spec.Vault.Name,
				Type:    t,
				Options: inventory.Spec.Vault.Options,
			})
			if err != nil {
				return err
			}
		}
		im.vault = v
	}

	// determine the vault to use for accessing credentials
	im.resetVault()

	return nil
}

func (im *inventoryManager) SetCredentialQuery(query string) error {
	qr, err := credentialquery.NewCredentialQueryRunner(query)
	if err != nil {
		return err
	}
	im.credentialQueryRunner = qr
	return nil
}

func (im *inventoryManager) GetAssets() []*asset.Asset {
	// TODO: do we need additional work to make this thread-safe
	return im.assetList
}

func (im *inventoryManager) GetRelatedAssets() []*asset.Asset {
	return im.relatedAssets
}

// QuerySecretId provides an input and determines the credential information for an asset
// The credential will only include the reference to the secret and not include the actual secret
func (im *inventoryManager) QuerySecretId(a *asset.Asset) (*vault.Credential, error) {
	if im.credentialQueryRunner == nil {
		log.Debug().Msg("no credential query set")
		return nil, nil
	}

	// this is where we get the vault configuration query and evaluate it against the asset data
	// if vault and secret function is set, run the additional handling
	return im.credentialQueryRunner.Run(a)
}

func (im *inventoryManager) Resolve(ctx context.Context) map[*asset.Asset]error {
	resolvedAssets := discovery.ResolveAssets(ctx, im.assetList, im.GetCredsResolver(), im.QuerySecretId)

	// TODO: iterate over all resolved assets and match them with the original list and try to find credentials for each asset
	im.assetList = resolvedAssets.Assets
	im.relatedAssets = resolvedAssets.RelatedAssets

	log.Info().Int("resolved-assets", len(im.assetList)).Msg("resolved assets")
	logger.DebugDumpJSON("inventory-resolved-assets", im.assetList)
	return resolvedAssets.Errors
}

func (im *inventoryManager) GetCredsResolver() vault.Resolver {
	return credentials_resolver.New(im.accessVault, im.isCached)
}

func (im *inventoryManager) GetCredential(cred *vault.Credential) (*vault.Credential, error) {
	return im.GetCredsResolver().GetCredential(cred)
}

func (im *inventoryManager) resetVault() {
	if im.vault != nil && im.inmemoryVault != nil {
		im.accessVault = multivault.New(im.vault, im.inmemoryVault)
	} else if im.vault != nil {
		im.accessVault = im.vault
	} else if im.inmemoryVault != nil {
		im.accessVault = im.inmemoryVault
	} else {
		im.accessVault = nil
	}
}

func (im *inventoryManager) GetVault() vault.Vault {
	return im.accessVault
}
