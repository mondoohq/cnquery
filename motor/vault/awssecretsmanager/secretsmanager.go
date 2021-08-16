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
	// create the client
	parsedArn, err := arn.Parse(id.Key)
	if err != nil {
		return nil, err
	}
	cfg := v.cfg.Copy()
	cfg.Region = parsedArn.Region
	c := secretsmanager.NewFromConfig(cfg)

	// retrieve secret
	log.Debug().Str("secret-id", id.Key).Msg("getting cred from aws secrets manager")
	out, err := c.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &id.Key,
	})
	if err != nil {
		log.Debug().Err(err).Str("secret-id", id.Key).Msg("could not retrieve secret from aws secret manager")
		return nil, vault.NotFoundError
	}

	// NOTE: we cannot use out.SecretBinary since it is not guaranteed to be set
	var data []byte
	if out.SecretString != nil {
		data = []byte(*out.SecretString)
	} else {
		data = out.SecretBinary
	}

	return &vault.Secret{
		Key:  id.Key,
		Data: data,
	}, nil
}

func (v *Vault) Set(ctx context.Context, cred *vault.Secret) (*vault.SecretID, error) {
	return nil, errors.New("not implemented")
}
