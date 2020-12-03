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

func (v *Vault) Set(ctx context.Context, cred *vault.Credential) (*vault.CredentialID, error) {
	err := validKey(cred.Key)
	if err != nil {
		return nil, err
	}

	// create the client
	c, err := v.client(ctx)
	if err != nil {
		return nil, err
	}

	// check if the secret already exists
	secret, err := c.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s", v.projectID, gcpKeyID(cred.Key)),
	})
	if err != nil {
		// create a new secret
		secret, err = c.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
			Parent:   fmt.Sprintf("projects/%s", v.projectID),
			SecretId: gcpKeyID(cred.Key),
			Secret: &secretmanagerpb.Secret{
				Replication: &secretmanagerpb.Replication{
					Replication: &secretmanagerpb.Replication_Automatic_{
						Automatic: &secretmanagerpb.Replication_Automatic{},
					},
				},
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create secret")
		}
	}

	// we json-encode the value, while proto would be more efficient, json allows humans to read the data more easily
	payload, err := json.Marshal(cred.Fields)
	if err != nil {
		return nil, err
	}

	// store new secret version
	_, err = c.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: secret.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: payload,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to add secret version")
	}

	return &vault.CredentialID{Key: cred.Key}, nil

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

	return &vault.Credential{
		Key: id.Key,
		// TODO: add label support
		// Label:  i.Label,
		Fields: fields,
	}, nil
}

func (v *Vault) Delete(ctx context.Context, id *vault.CredentialID) (*vault.CredentialDeletedResp, error) {
	return nil, notImplemented
}
