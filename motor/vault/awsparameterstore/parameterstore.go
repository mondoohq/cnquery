package awsparameterstore

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/motor/vault"
)

var notImplemented = errors.New("not implemented")

// https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html
// https://docs.aws.amazon.com/systems-manager/latest/APIReference/API_GetParameter.html
func New(cfg aws.Config) *Vault {
	return &Vault{
		cfg: cfg,
	}
}

type Vault struct {
	cfg aws.Config
}

// we need to remove the leading // from mrns, this should not be done here, therefore we just throw an error
func validKey(key string) error {
	if strings.HasPrefix(key, "/") {
		return errors.New("leading / are not allowed")
	}
	return nil
}

func awsParamKeyID(key string) string {
	gcpKey := strings.ReplaceAll(key, "/", "-")
	gcpKey = strings.ReplaceAll(gcpKey, ".", "-")
	return gcpKey
}

func (v *Vault) Get(ctx context.Context, id *vault.SecretID) (*vault.Secret, error) {
	err := validKey(id.Key)
	if err != nil {
		return nil, err
	}

	// create the client
	c := ssm.NewFromConfig(v.cfg)

	// retrieve secret
	out, err := c.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(awsParamKeyID(id.Key)),
		WithDecryption: true,
	})
	if err != nil {
		return nil, vault.NotFoundError
	}

	var data []byte
	if out != nil && out.Parameter != nil {
		v := *out.Parameter.Value
		data = []byte(v)
	}

	return &vault.Secret{
		Key:  id.Key,
		Data: data,
	}, nil
}

func (v *Vault) Set(ctx context.Context, cred *vault.Secret) (*vault.SecretID, error) {
	return nil, errors.New("not implemented")
}
