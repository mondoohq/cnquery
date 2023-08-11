package connection

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
)

type AwsConnection struct {
	id          uint32
	Conf        *inventory.Config
	asset       *inventory.Asset
	cfg         aws.Config
	accountId   string
	clientcache ClientsCache
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
	svc := sts.NewFromConfig(cfg)
	resp, err := svc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	} else if resp.Account == nil || resp.UserId == nil {
		return nil, errors.New("could not read iam user")
	} else {
		return resp, nil
	}
}

func (h *AwsConnection) Regions() ([]string, error) {
	// check cache for regions list, return if exists
	c, ok := h.clientcache.Load("_regions")
	if ok {
		log.Debug().Msg("use regions from cache")
		return c.Data.([]string), nil
	}
	log.Debug().Msg("no region cache found. fetching regions")

	// if no cache, get regions using ec2 client (using the ssm list global regions does not give the same list)
	regions := []string{}
	svc := h.Ec2("us-east-1")
	ctx := context.Background()

	res, err := svc.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		// try with govcloud region
		svc := h.Ec2("us-gov-west-1")
		res, err = svc.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
		if err != nil {
			return regions, err
		}
	}
	for _, region := range res.Regions {
		regions = append(regions, *region.RegionName)
	}
	// cache the regions as part of the provider instance
	h.clientcache.Store("_regions", &CacheEntry{Data: regions})
	return regions, nil
}
