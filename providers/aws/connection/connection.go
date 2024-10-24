// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
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
	scope             string
}

type DiscoveryFilters struct {
	Ec2DiscoveryFilters Ec2DiscoveryFilters
	EcrDiscoveryFilters EcrDiscoveryFilters
	EcsDiscoveryFilters EcsDiscoveryFilters
	DiscoveryFilters    GeneralDiscoveryFilters
}

// ensure all underlying reference types aren't `nil`
func EmptyDiscoveryFilters() DiscoveryFilters {
	return DiscoveryFilters{
		DiscoveryFilters:    GeneralDiscoveryFilters{Regions: []string{}, ExcludeRegions: []string{}},
		Ec2DiscoveryFilters: Ec2DiscoveryFilters{InstanceIds: []string{}, ExcludeInstanceIds: []string{}, Tags: map[string]string{}, ExcludeTags: map[string]string{}},
		EcrDiscoveryFilters: EcrDiscoveryFilters{Tags: []string{}, ExcludeTags: []string{}},
		EcsDiscoveryFilters: EcsDiscoveryFilters{},
	}
}

type GeneralDiscoveryFilters struct {
	Regions        []string
	ExcludeRegions []string
}

type Ec2DiscoveryFilters struct {
	InstanceIds        []string
	ExcludeInstanceIds []string
	Tags               map[string]string
	ExcludeTags        map[string]string
}

type EcrDiscoveryFilters struct {
	Tags        []string
	ExcludeTags []string
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
		Filters:          EmptyDiscoveryFilters(),
	}
	opts := parseFlagsForConnectionOptions(asset.Options, conf.Options, conf.GetCredentials())
	for _, opt := range opts {
		opt(c)
	}
	// custom retry client
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 5
	retryClient.Logger = &zeroLogAdapter{}
	c.awsConfigOptions = append(c.awsConfigOptions, config.WithHTTPClient(retryClient.StandardClient()))

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
		log.Warn().Interface("opts", conf.Discover.Filter).Msg("PARSING OPTS TO FILTERS!")
		c.Filters = parseOptsToFilters(conf.Discover.Filter)
	}
	return c, nil
}

func parseOptsToFilters(opts map[string]string) DiscoveryFilters {
	d := EmptyDiscoveryFilters()
	for k, v := range opts {
		switch {
		case k == "regions":
			d.DiscoveryFilters.Regions = append(d.DiscoveryFilters.Regions, strings.Split(v, ",")...)
		case k == "exclude:regions":
			d.DiscoveryFilters.ExcludeRegions = append(d.DiscoveryFilters.ExcludeRegions, strings.Split(v, ",")...)
		case k == "ec2:iid":
			d.Ec2DiscoveryFilters.InstanceIds = append(d.Ec2DiscoveryFilters.InstanceIds, strings.Split(v, ",")...)
		case k == "ec2:exclude:iid":
			d.Ec2DiscoveryFilters.ExcludeInstanceIds = append(d.Ec2DiscoveryFilters.ExcludeInstanceIds, strings.Split(v, ",")...)
		case strings.HasPrefix(k, "ec2:tag:"):
			d.Ec2DiscoveryFilters.Tags[strings.TrimPrefix(k, "ec2:tag:")] = v
		case strings.HasPrefix(k, "ec2:exclude:tag:"):
			d.Ec2DiscoveryFilters.ExcludeTags[strings.TrimPrefix(k, "ec2:exclude:tag:")] = v
		case k == "ecr:tags":
			d.EcrDiscoveryFilters.Tags = append(d.EcrDiscoveryFilters.Tags, strings.Split(v, ",")...)
		case k == "ecr:exclude:tags":
			d.EcrDiscoveryFilters.ExcludeTags = append(d.EcrDiscoveryFilters.ExcludeTags, strings.Split(v, ",")...)
		case k == "ecs:only-running-containers":
			parsed, err := strconv.ParseBool(v)
			if err == nil {
				d.EcsDiscoveryFilters.OnlyRunningContainers = parsed
			}
		case k == "ecs:discover-instances":
			parsed, err := strconv.ParseBool(v)
			if err == nil {
				d.EcsDiscoveryFilters.DiscoverInstances = parsed
			}
		case k == "ecs:discover-images":
			parsed, err := strconv.ParseBool(v)
			if err == nil {
				d.EcsDiscoveryFilters.DiscoverImages = parsed
			}
		}
	}
	return d
}

func parseFlagsForConnectionOptions(m1 map[string]string, m2 map[string]string, creds []*vault.Credential) []ConnectionOption {
	// merge the options to make sure we dont miss anything
	m := m1
	if m == nil {
		m = make(map[string]string)
	}
	for k, v := range m2 {
		m[k] = v
	}
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

	if len(creds) > 0 {
		cred := creds[0]
		o = append(o, WithStaticCredentials(m["access-key-id"], string(cred.Secret), m["session-token"]))
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

func WithStaticCredentials(key string, secret string, token string) ConnectionOption {
	return func(a *AwsConnection) {
		a.awsConfigOptions = append(a.awsConfigOptions, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(key, secret, token)))
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

func (p *AwsConnection) UpdateAsset(asset *inventory.Asset) {
	p.asset = asset
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

	// include filters have precedense over exclude filters. in any normal situation they should be mutually exclusive.
	regionLimits := h.Filters.DiscoveryFilters.Regions
	log.Warn().Interface("regionLimits", regionLimits).Msg("region limits when Regions() is called")
	if len(regionLimits) > 0 {
		log.Debug().Interface("regions", regionLimits).Msg("using region limits")
		// cache the regions as part of the provider instance
		h.clientcache.Store("_regions", &CacheEntry{Data: regionLimits})
		return regionLimits, nil
	}
	// if no cache, get regions using ec2 client (using the ssm list global regions does not give the same list)
	log.Debug().Msg("no region cache or region limits found. fetching regions")
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
		// ensure excluded regions are discarded
		if !slices.Contains(h.Filters.DiscoveryFilters.ExcludeRegions, *region.RegionName) {
			regions = append(regions, *region.RegionName)
		}
	}
	// cache the regions as part of the provider instance
	h.clientcache.Store("_regions", &CacheEntry{Data: regions})
	return regions, nil
}

// zeroLogAdapter is the adapter for retryablehttp is outputting debug messages
type zeroLogAdapter struct{}

func (l *zeroLogAdapter) Msg(msg string, keysAndValues ...interface{}) {
	var e *zerolog.Event
	// retry messages should only go to debug
	e = log.Debug()
	for i := 0; i < len(keysAndValues); i += 2 {
		e = e.Interface(keysAndValues[i].(string), keysAndValues[i+1])
	}
	e.Msg(msg)
}

func (l *zeroLogAdapter) Error(msg string, keysAndValues ...interface{}) {
	l.Msg(msg, keysAndValues...)
}

func (l *zeroLogAdapter) Info(msg string, keysAndValues ...interface{}) {
	l.Msg(msg, keysAndValues...)
}

func (l *zeroLogAdapter) Debug(msg string, keysAndValues ...interface{}) {
	l.Msg(msg, keysAndValues...)
}

func (l *zeroLogAdapter) Warn(msg string, keysAndValues ...interface{}) {
	l.Msg(msg, keysAndValues...)
}
