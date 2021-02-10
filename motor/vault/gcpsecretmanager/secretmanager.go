package gcpsecretmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/pkg/errors"
	"go.mondoo.io/mondoo/motor/vault"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var notImplemented = errors.New("not implemented")

// https://cloud.google.com/secret-manager
// https://cloud.google.com/secret-manager/docs/reference/libraries#client-libraries-install-go
func New(projectID string) *Vault {
	return &Vault{
		projectID: projectID,
	}
}

type Vault struct {
	projectID string
}

// Dial gets a Vault client.
func (v *Vault) client(ctx context.Context) (*secretmanager.Client, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup gcp secret manager client")
	}
	return client, nil
}

// we need to remove the leading // from mrns, this should not be done here, therefore we just throw an error
func validKey(key string) error {
	if strings.HasPrefix(key, "/") {
		return errors.New("leading / are not allowed")
	}
	return nil
}

// gcp does not support / in strings which we heavily use
// 😤 secret names can only contain english letters (A-Z), numbers (0-9), dashes (-), and underscores (_)
// therefore we cannot use url encode and we need to fallback to an unsafe mechanism where we may
// run into issues of two keys matching the same value, lets not maintain a mapping table for now
// since we do not allow "list" a one-way transformation is okay for now
func gcpKeyID(key string) string {
	gcpKey := strings.ReplaceAll(key, "/", "-")
	gcpKey = strings.ReplaceAll(gcpKey, ".", "-")
	return gcpKey
}

func (v *Vault) Get(ctx context.Context, id *vault.CredentialID) (*vault.Credential, error) {
	err := validKey(id.Key)
	if err != nil {
		return nil, err
	}

	c, err := v.client(ctx)
	if err != nil {
		return nil, err
	}

	// retrieve secret metadata
	result, err := c.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", v.projectID, gcpKeyID(id.Key)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to access secret version")
	}

	var fields map[string]string
	err = json.Unmarshal(result.Payload.Data, &fields)
	if err != nil {
		return nil, err
	}
	var data string
	if result != nil && result.Payload != nil {
		data = string(result.Payload.Data)
	}

	return &vault.Credential{
		Key:    id.Key,
		Secret: data,
	}, nil
}

func (v *Vault) Set(ctx context.Context, cred *vault.Credential) (*vault.CredentialID, error) {
	return nil, errors.New("not implemented")
}
