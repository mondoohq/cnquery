package keyring

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"

	"github.com/99designs/keyring"
	"go.mondoo.io/mondoo/motor/vault"
)

var notImplemented = errors.New("not implemented")

func New(serviceName string) *Vault {
	return &Vault{
		ServiceName: serviceName,
		fileDir:     "~/.mondoo/",
		filePasswordFunc: func(s string) (string, error) {
			// TODO: this only applies to cases where we have no real keychain available
			return "", errors.New("file-fallback is not supported")
		},
		// by default we do not allow a fallback to encrypted keys
		allowedBackends: []keyring.BackendType{
			// Windows
			keyring.WinCredBackend,
			// MacOS
			keyring.KeychainBackend,
			// Linux
			keyring.SecretServiceBackend,
			keyring.KWalletBackend,
			// General
			keyring.PassBackend,
		},
	}
}

func NewEncryptedFile(path string, serviceName string, password string) *Vault {
	return &Vault{
		ServiceName: serviceName,
		fileDir:     path,
		filePasswordFunc: func(s string) (string, error) {
			return password, nil
		},
		allowedBackends: []keyring.BackendType{
			keyring.FileBackend,
		},
	}
}

type Vault struct {
	ServiceName      string
	allowedBackends  []keyring.BackendType
	fileDir          string
	filePasswordFunc func(s string) (string, error)
}

func (v *Vault) open() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName:      v.ServiceName,
		AllowedBackends:  v.allowedBackends,
		FileDir:          v.fileDir,
		FilePasswordFunc: v.filePasswordFunc,
	})
}

func (v *Vault) Set(ctx context.Context, cred *vault.Secret) (*vault.SecretID, error) {
	ring, err := v.open()
	if err != nil {
		return nil, err
	}

	// TODO: store data as json encoding
	err = ring.Set(keyring.Item{
		Key:   cred.Key,
		Label: cred.Label,
		Data:  cred.Data,
	})

	return &vault.SecretID{
		Key: cred.Key,
	}, err
}

func (v *Vault) Get(ctx context.Context, id *vault.SecretID) (*vault.Secret, error) {
	ring, err := v.open()
	if err != nil {
		return nil, err
	}

	i, err := ring.Get(id.Key)
	if err != nil {
		log.Debug().Err(err).Msg("could not retrieve secret from keyring")
		return nil, vault.NotFoundError
	}

	return &vault.Secret{
		Key:   i.Key,
		Label: i.Label,
		Data:  i.Data,
	}, nil
}
