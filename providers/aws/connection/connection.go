package connection

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
)

type AwsConnection struct {
	id        uint32
	Conf      *inventory.Config
	asset     *inventory.Asset
	cfg       aws.Config
	accountId string
}

func NewAwsConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AwsConnection, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	if cfg.Region == "" {
		log.Info().Msg("no AWS region found, using us-east-1")
		cfg.Region = "us-east-1" // in case the user has no region set, default to us-east-1
	}
	asset.Platform = &inventory.Platform{
		Title: "aws",
		Name:  "aws",
	}
	// gather information about the aws account
	cfgCopy := cfg.Copy()
	identity, err := CheckIam(cfgCopy)
	if err != nil {
		log.Debug().Err(err).Msg("could not gather details of AWS account")
		// try with govcloud region
		cfgCopy.Region = "us-gov-west-1"
		identity, err = CheckIam(cfgCopy)
		if err != nil {
			log.Debug().Err(err).Msg("could not gather details of AWS account")
			return nil, err
		}
	}
	return &AwsConnection{
		Conf:      conf,
		id:        id,
		asset:     asset,
		cfg:       cfg,
		accountId: *identity.Account,
	}, nil
}

func (h *AwsConnection) Name() string {
	return "aws"
}

func (h *AwsConnection) ID() uint32 {
	return h.id
}

func (p *AwsConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *AwsConnection) AccountId() string {
	return p.accountId
}

func CheckIam(cfg aws.Config) (*sts.GetCallerIdentityOutput, error) {
	ctx := context.Background()
	stsSvr := sts.NewFromConfig(cfg)
	resp, err := stsSvr.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	} else if resp.Account == nil || resp.UserId == nil {
		return nil, errors.New("could not read iam user")
	} else {
		return resp, nil
	}
}
