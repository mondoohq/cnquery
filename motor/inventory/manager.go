package inventory

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/logger"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery"
	"go.mondoo.io/mondoo/motor/inventory/v1"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/vault"
	"go.mondoo.io/mondoo/motor/vault/inmemory"
	"google.golang.org/protobuf/proto"
)

// TODO: we may want to use a proto service for this implementation
type CredentialManager interface {
	GetCredential(secretId string) (*transports.Credential, error)
}

type InventoryManager interface {
	GetAssets() []*asset.Asset
	GetCredential(secretId string) (*transports.Credential, error)
	GetVault() vault.Vault
}

type Option func(*inventoryManager) error

// passes a pre-parsed asset inventory into the Inventory Manager
func WithInventory(inventory *v1.Inventory) Option {
	return func(im *inventoryManager) error {
		logger.DebugDumpJSON("mondoo-inventory", inventory)
		return im.loadInventory(inventory)
	}
}

// passes a list of asset into the Inventory Manager
func WithAssets(assetList []*asset.Asset) Option {
	return func(im *inventoryManager) error {
		inventory := &v1.Inventory{
			Spec: &v1.InventorySpec{
				Assets: assetList,
			},
		}
		return im.loadInventory(inventory)
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

	return im, nil
}

type inventoryManager struct {
	assetList []*asset.Asset
	vault     vault.Vault
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

		secret, err := NewSecret(cred)
		if err != nil {
			return err
		}

		secrets[secret.Key] = secret
	}

	im.vault = inmemory.New(inmemory.WithSecretMap(secrets))

	// TODO: use multi-vault to warp the provided vault and all inline-configured credentials
	return nil
}

func (im *inventoryManager) GetAssets() []*asset.Asset {
	// TODO: do we need additional work to make this thread-safe
	return im.assetList
}

func (im *inventoryManager) GetCredential(secretId string) (*transports.Credential, error) {
	secret, err := im.vault.Get(context.Background(), &vault.SecretID{
		Key: secretId,
	})
	if err != nil {
		return nil, err
	}

	return NewCredential(secret)
}

func (im *inventoryManager) Resolve() map[*asset.Asset]error {
	resolvedAssets := discovery.ResolveAssets(im.assetList)

	// TODO: iterate over all resolved assets and match them with the original list and try to find credentials for each asset
	im.assetList = resolvedAssets.Assets

	log.Info().Int("resolved-assets", len(im.assetList)).Msg("resolved assets")
	return resolvedAssets.Errors
}

func (im *inventoryManager) GetVault() vault.Vault {
	return im.vault
}

func NewSecret(cred *transports.Credential) (*vault.Secret, error) {
	// TODO: we also encode the ID, this may not be a good approach
	secretData, err := proto.Marshal(cred)
	if err != nil {
		return nil, err
	}

	return &vault.Secret{
		Key:  cred.SecretId,
		Data: secretData, // TODO: rename secret.secret into Data
	}, nil
}

func NewCredential(sec *vault.Secret) (*transports.Credential, error) {
	var cred transports.Credential
	err := proto.Unmarshal(sec.Data, &cred)
	if err != nil {
		return nil, err
	}
	cred.SecretId = sec.Key
	return &cred, nil
}
