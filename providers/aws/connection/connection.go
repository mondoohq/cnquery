// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

type AwsConnection struct {
	plugin.Connection
	Conf              *inventory.Config
	asset             *inventory.Asset
	cfg               aws.Config
	accountId         string
	clientcache       ClientsCache
	awsConfigOptions  []func(*config.LoadOptions) error
	profile           string
	PlatformOverride  string
	connectionOptions map[string]string
	Filters           DiscoveryFilters
	RegionLimits      []string
	scope             string
}

type DiscoveryFilters struct {
	Ec2DiscoveryFilters     Ec2DiscoveryFilters
	EcrDiscoveryFilters     EcrDiscoveryFilters
	EcsDiscoveryFilters     EcsDiscoveryFilters
	GeneralDiscoveryFilters GeneralResourceDiscoveryFilters
}

type GeneralResourceDiscoveryFilters struct {
	Tags    map[string]string
	Regions []string
}

type Ec2DiscoveryFilters struct {
	Regions     []string
	Tags        map[string]string
	InstanceIds []string
}
type EcrDiscoveryFilters struct {
	Tags []string
}
type EcsDiscoveryFilters struct {
	OnlyRunningContainers bool
	DiscoverImages        bool
	DiscoverInstances     bool
}

func NewMockConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) *AwsConnection {
	return &AwsConnection{
		Connection: plugin.NewConnection(id, asset),
		asset:      asset,
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

	c.Connection = plugin.NewConnection(id, asset)
	c.Conf = conf
	c.asset = asset
	c.cfg = cfg
	c.accountId = *identity.Account
	c.profile = asset.Options["profile"]
	c.scope = asset.Options["scope"]
	c.connectionOptions = asset.Options
	if conf.Discover != nil {
		c.Filters = parseOptsToFilters(conf.Discover.Filter)
		c.RegionLimits = c.Filters.GeneralDiscoveryFilters.Regions
	}
	return c, nil
}

func parseOptsToFilters(opts map[string]string) DiscoveryFilters {
	d := DiscoveryFilters{
		Ec2DiscoveryFilters:     Ec2DiscoveryFilters{Tags: map[string]string{}},
		EcsDiscoveryFilters:     EcsDiscoveryFilters{},
		EcrDiscoveryFilters:     EcrDiscoveryFilters{Tags: []string{}},
		GeneralDiscoveryFilters: GeneralResourceDiscoveryFilters{Tags: map[string]string{}},
	}
	for k, v := range opts {
		switch {
		case strings.HasPrefix(k, "ec2:tag:"):
			d.Ec2DiscoveryFilters.Tags[strings.TrimPrefix(k, "ec2:tag:")] = v
		case k == "ec2:region":
			d.Ec2DiscoveryFilters.Regions = append(d.Ec2DiscoveryFilters.Regions, v)
		case k == "all:region", k == "region":
			d.GeneralDiscoveryFilters.Regions = append(d.GeneralDiscoveryFilters.Regions, v)
		case k == "instance-id":
			d.Ec2DiscoveryFilters.InstanceIds = append(d.Ec2DiscoveryFilters.InstanceIds, v)
		case strings.HasPrefix(k, "all:tag:"):
			d.GeneralDiscoveryFilters.Tags[strings.TrimPrefix(k, "all:tag:")] = v
		case k == "ecr:tag":
			d.EcrDiscoveryFilters.Tags = append(d.EcrDiscoveryFilters.Tags, v)
		}
	}
	return d
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

func (p *AwsConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *AwsConnection) AccountId() string {
	return p.accountId
}

func (p *AwsConnection) Profile() string {
	return p.profile
}

func (p *AwsConnection) Scope() string {
	return p.scope
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

const MISSING_REGION_MSG = `The AWS region must be set for the deployment. Please use environment variables
or AWS profiles. Further details are available at:
- https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html
- https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html`

// CheckRegion verifies that the config includes a region
func CheckRegion(cfg aws.Config) error {
	if len(cfg.Region) == 0 {
		return errors.New(MISSING_REGION_MSG)
	}
	return nil
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

	if len(h.RegionLimits) > 0 {
		log.Debug().Interface("regions", h.RegionLimits).Msg("using region limits")
		// cache the regions as part of the provider instance
		h.clientcache.Store("_regions", &CacheEntry{Data: h.RegionLimits})
		return h.RegionLimits, nil
	}
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
