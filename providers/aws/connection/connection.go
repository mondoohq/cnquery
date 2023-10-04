// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
)

type AwsConnection struct {
	id                uint32
	Conf              *inventory.Config
	asset             *inventory.Asset
	cfg               aws.Config
	accountId         string
	clientcache       ClientsCache
	awsConfigOptions  []func(*config.LoadOptions) error
	profile           string
	PlatformOverride  string
	connectionOptions map[string]string
}

func NewMockConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) *AwsConnection {
	return &AwsConnection{
		id:    id,
		asset: asset,
	}
}

func NewAwsConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AwsConnection, error) {
	log.Debug().Msg("new aws connection")
	// check flags for connection options
	c := &AwsConnection{
		awsConfigOptions: []func(*config.LoadOptions) error{},
	}
	opts := parseFlagsForConnectionOptions(asset.Options)
	for _, opt := range opts {
		opt(c)
	}
	cfg, err := config.LoadDefaultConfig(context.Background(), c.awsConfigOptions...)
	if err != nil {
		return nil, err
	}
	if cfg.Region == "" {
		log.Info().Msg("no AWS region found, using us-east-1")
		cfg.Region = "us-east-1" // in case the user has no region set, default to us-east-1
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

	c.Conf = conf
	c.id = id
	c.asset = asset
	c.cfg = cfg
	c.accountId = *identity.Account
	c.profile = asset.Options["profile"]
	c.connectionOptions = asset.Options
	return c, nil
}

func parseFlagsForConnectionOptions(m map[string]string) []ConnectionOption {
	o := make([]ConnectionOption, 0)
	if apiEndpoint, ok := m["endpoint-url"]; ok {
		o = append(o, WithEndpoint(apiEndpoint))
	}

	if awsRegion, ok := m["region"]; ok {
		log.Debug().Str("region", awsRegion).Msg("using region")
		o = append(o, WithRegion(awsRegion))
	}

	if awsProfile, ok := m["profile"]; ok {
		log.Debug().Str("profile", awsProfile).Msg("using aws profile")
		o = append(o, WithProfile(awsProfile))
	}

	if role, ok := m["role"]; ok {
		log.Debug().Str("role", role).Msg("using aws sts assume role")
		cfg, _ := config.LoadDefaultConfig(context.Background())
		externalId := m["external-id"]
		o = append(o, WithAssumeRole(cfg, role, externalId))
	}
	return o
}

type ConnectionOption func(charp *AwsConnection)

// // delegate back to the default v2 resolver otherwise
// return s3.NewDefaultEndpointResolverV2().ResolveEndpoint(ctx, params)
func WithEndpoint(apiEndpoint string) ConnectionOption {
	return func(a *AwsConnection) {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if apiEndpoint != "" {
				return aws.Endpoint{
					PartitionID:   "aws",
					URL:           apiEndpoint,
					SigningRegion: region,
				}, nil
			}

			// returning EndpointNotFoundError will allow the service to fallback to its default resolution
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		a.awsConfigOptions = append(a.awsConfigOptions, config.WithEndpointResolverWithOptions(customResolver))
	}
}

func WithRegion(region string) ConnectionOption {
	return func(a *AwsConnection) {
		a.awsConfigOptions = append(a.awsConfigOptions, config.WithRegion(region))
	}
}

func WithProfile(profile string) ConnectionOption {
	return func(a *AwsConnection) {
		a.awsConfigOptions = append(a.awsConfigOptions, config.WithSharedConfigProfile(profile))
	}
}

func WithExternalId(id string) func(o *stscreds.AssumeRoleOptions) {
	if id != "" {
		return func(o *stscreds.AssumeRoleOptions) {
			o.ExternalID = &id
		}
	}
	return func(o *stscreds.AssumeRoleOptions) {}
}

func WithAssumeRole(defaultCfg aws.Config, roleArn string, externalId string) ConnectionOption {
	opts := WithExternalId(externalId)
	return func(a *AwsConnection) {
		stsClient := sts.NewFromConfig(defaultCfg)
		a.awsConfigOptions = append(a.awsConfigOptions, config.WithCredentialsProvider(
			aws.NewCredentialsCache(
				stscreds.NewAssumeRoleProvider(
					stsClient,
					roleArn,
					opts,
				)),
		))
	}
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

func (p *AwsConnection) Profile() string {
	return p.profile
}

func (p *AwsConnection) ConnectionOptions() map[string]string {
	return p.connectionOptions
}

func (p *AwsConnection) RunCommand(command string) (*shared.Command, error) {
	return nil, errors.New("unimplemented")
}

func (p *AwsConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	return shared.FileInfoDetails{}, errors.New("unimplemented")
}

func (p *AwsConnection) FileSystem() afero.Fs {
	return nil
}

func (p *AwsConnection) Capabilities() shared.Capabilities {
	return shared.Capability_RunCommand // not true, update to nothing
}

func (p *AwsConnection) Type() shared.ConnectionType {
	return "aws"
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
