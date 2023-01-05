package gcpberglas

import (
	"context"
	"strings"

	berglas "github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/motor/vault"
)

// https://github.com/GoogleCloudPlatform/berglas
func New(projectID string) *Vault {
	return &Vault{
		projectID: projectID,
	}
}

// should be instantiated with NewWithKey if vault.Set is to be used
func NewWithKey(projectID string, kmsKeyID *string) *Vault {
	return &Vault{
		projectID: projectID,
		kmsKeyID:  kmsKeyID,
	}
}

type berglasStorageInfo struct {
	bucket string
	object string
}

type Vault struct {
	projectID string
	kmsKeyID  *string
}

func (v *Vault) About(context.Context, *vault.Empty) (*vault.VaultInfo, error) {
	return &vault.VaultInfo{Name: "GCP Berglas: " + v.projectID}, nil
}

// Dial gets a Vault client.
func (v *Vault) client(ctx context.Context) (*berglas.Client, error) {
	client, err := berglas.New(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup gcp berglas client")
	}
	return client, nil
}

// expected berglas key format: storage/{bucketName}/{objectName}
func getBerglasStorageInfo(key string) (berglasStorageInfo, error) {
	split := strings.Split(key, "/")
	if len(split) != 3 {
		return berglasStorageInfo{}, errors.New("invalid berglas key provided")
	}
	// we omit storage for now, as berglas secrets manager is not yet supported
	// however, the key contains the type as to not break any existing keys if we add support in the future
	return berglasStorageInfo{
		bucket: split[1],
		object: split[2],
	}, nil
}

func (v *Vault) Get(ctx context.Context, id *vault.SecretID) (*vault.Secret, error) {
	c, err := v.client(ctx)
	if err != nil {
		return nil, err
	}

	berglasReadInfo, err := getBerglasStorageInfo(id.Key)
	if err != nil {
		return nil, err
	}

	result, err := c.Read(ctx, &berglas.StorageReadRequest{
		Bucket: berglasReadInfo.bucket,
		Object: berglasReadInfo.object,
	})
	if err != nil {
		return nil, vault.NotFoundError
	}

	return &vault.Secret{
		Key:  id.Key,
		Data: result.Plaintext,
	}, nil
}

func (v *Vault) Set(ctx context.Context, cred *vault.Secret) (*vault.SecretID, error) {
	if v.kmsKeyID == nil {
		return nil, errors.New("cannot create vault secret without KMS key id")
	}
	if len(*v.kmsKeyID) == 0 {
		return nil, errors.New("specified KMS key id is empty")
	}
	c, err := v.client(ctx)
	if err != nil {
		return nil, err
	}

	berglasReadInfo, err := getBerglasStorageInfo(cred.Key)
	if err != nil {
		return nil, err
	}

	_, err = c.Create(ctx, &berglas.StorageCreateRequest{
		Bucket:    berglasReadInfo.bucket,
		Object:    berglasReadInfo.object,
		Plaintext: cred.Data,
		Key:       *v.kmsKeyID,
	})
	if err != nil {
		return nil, err
	}
	return &vault.SecretID{
		Key: cred.Key,
	}, nil
}
