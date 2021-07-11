package awssecretsmanager

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/vault"
)

var notImplemented = errors.New("not implemented")

func New(cfg aws.Config) *Vault {
	return &Vault{
		cfg: cfg.Copy(),
	}
}

type Vault struct {
	cfg aws.Config
}

func (v *Vault) Get(ctx context.Context, id *vault.SecretID) (*vault.Secret, error) {
	log.Debug().Msgf("getting cred from aws secrets manager %s", id.Key)
	// create the client
	parsedArn, err := arn.Parse(id.Key)
	if err != nil {
		return nil, err
	}
	cfg := v.cfg.Copy()
	cfg.Region = parsedArn.Region
	c := secretsmanager.NewFromConfig(cfg)
	// retrieve secret
	out, err := c.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &id.Key,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret")
	}
	return &vault.Secret{
		Key:    id.Key,
		Secret: out.SecretBinary,
	}, nil
}

func (v *Vault) Set(ctx context.Context, cred *vault.Secret) (*vault.SecretID, error) {
	return nil, errors.New("not implemented")
}
