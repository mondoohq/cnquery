package hashivault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
	"go.mondoo.com/cnquery/motor/vault"
)

var notImplemented = errors.New("not implemented")

func New(serverURL string, token string) *Vault {
	return &Vault{
		Token: token,
		APIConfig: api.Config{
			Address: serverURL,
		},
	}
}

type Vault struct {
	// Token is the access token the Vault client uses to talk to the server.
	// See https://www.vaultproject.io/docs/concepts/tokens.html for more
	// information.
	Token string
	// APIConfig is used to configure the creation of the client.
	APIConfig api.Config
}

func (v *Vault) About(context.Context, *vault.Empty) (*vault.VaultInfo, error) {
	return &vault.VaultInfo{Name: "Hashicorp Vault"}, nil
}

// Dial gets a Vault client.
func (v *Vault) client() (*api.Client, error) {
	c, err := api.NewClient(&v.APIConfig)
	if err != nil {
		return nil, err
	}
	if v.Token != "" {
		c.SetToken(v.Token)
	}
	return c, nil
}

func vaultSecretId(key string) string {
	base := "secret/data/"
	return base + key
}

// we need to remove the leading // from mrns, this should not be done here, therefore we just throw an error
func validKey(key string) error {
	if strings.HasPrefix(key, "/") {
		return errors.New("leading / are not allowed")
	}
	return nil
}

// https://learn.hashicorp.com/tutorials/vault/versioned-kv?in=vault/secrets-management#step-2-write-secrets
func (v *Vault) Get(ctx context.Context, id *vault.SecretID) (*vault.Secret, error) {
	c, err := v.client()
	if err != nil {
		return nil, err
	}

	err = validKey(id.Key)
	if err != nil {
		return nil, err
	}

	secret, err := c.Logical().Read(vaultSecretId(id.Key))
	if err != nil {
		return nil, vault.NotFoundError
	}

	secretBytes, err := secretData(secret)
	if err != nil {
		return nil, err
	}

	return &vault.Secret{
		Key:  id.Key,
		Data: secretBytes,
	}, nil
}

// secretData returns the map of metadata associated with the secret
func secretData(s *api.Secret) ([]byte, error) {
	if s == nil {
		return nil, nil
	}

	if s.Data == nil || (s.Data["data"] == nil) {
		return nil, nil
	}

	data, ok := s.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unable to convert data field to expected format")
	}

	// when we resolve the secret in motor/discovery/resolve.go, we unmarshal to map[string]string, so things should match!
	secretData := make(map[string]string, len(data))
	for k, v := range data {
		typed, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("unable to convert data value %v to string", v)
		}
		secretData[k] = typed
	}

	return json.Marshal(secretData)
}

func (v *Vault) Set(ctx context.Context, cred *vault.Secret) (*vault.SecretID, error) {
	return nil, errors.New("not implemented")
}
