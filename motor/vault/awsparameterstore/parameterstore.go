package awsparameterstore

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
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

func (v *Vault) Set(ctx context.Context, cred *vault.Credential) (*vault.CredentialID, error) {
	err := validKey(cred.Key)
	if err != nil {
		return nil, err
	}

	// create the client
	c := ssm.NewFromConfig(v.cfg)

	// we json-encode the value, while proto would be more efficient, json allows humans to read the data more easily
	payload, err := json.Marshal(cred.Fields)
	if err != nil {
		return nil, err
	}

	// store new secret version
	_, err = c.PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String(awsParamKeyID(cred.Key)),
		Value:     aws.String(string(payload)),
		Type:      types.ParameterTypeString,
		Overwrite: true,
		// NOTE: once we use  tags, override will not work
		// Tags:  []ssm.Tag{ssm.Tag{Key: aws.String("key"), Value: aws.String("value")}},
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

	// create the client
	c := ssm.NewFromConfig(v.cfg)

	// retrieve secret
	out, err := c.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(awsParamKeyID(id.Key)),
		WithDecryption: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret")
	}

	var data []byte
	if out != nil {
		data = []byte(*out.Parameter.Value)
	}

	var fields map[string]string
	err = json.Unmarshal(data, &fields)
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
