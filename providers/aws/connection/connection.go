// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/account"
	"github.com/aws/aws-sdk-go-v2/service/account/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/logger/zerologadapter"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

type AwsConnection struct {
	plugin.Connection
	Conf             *inventory.Config
	asset            *inventory.Asset
	cfg              aws.Config
	accountId        string
	clientcache      ClientsCache
	awsConfigOptions []func(*config.LoadOptions) error
	PlatformOverride string
	Filters          DiscoveryFilters

	opts awsConnectionOptions
}

type awsConnectionOptions struct {
	scope   string
	profile string
	options map[string]string
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

	// merge the options to make sure we don't miss anything
	if asset.Options == nil {
		asset.Options = map[string]string{}
	}
	maps.Copy(asset.Options, conf.Options)

	opts := parseFlagsForConnectionOptions(asset.Options, conf.GetCredentials())
	for _, opt := range opts {
		opt(c)
	}
	// custom retry client with reduced retries and shorter backoff
	// to avoid excessive delays when regions are unreachable
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 2                    // reduced from 5 to avoid long delays on unreachable regions
	retryClient.RetryWaitMax = 10 * time.Second // cap at 10s instead of 30s
	retryClient.Logger = zerologadapter.New(log.Logger)
	c.awsConfigOptions = append(c.awsConfigOptions, config.WithHTTPClient(retryClient.StandardClient()))

	cfg, err := config.LoadDefaultConfig(context.Background(), c.awsConfigOptions...)
	if err != nil {
		return nil, err
	}
	if cfg.Region == "" {
		log.Info().Msg("no AWS region found, using us-east-1")
		cfg.Region = "us-east-1" // in case the user has no region set, default to us-east-1
	}

	c.Connection = plugin.NewConnection(id, asset)
	c.Conf = conf
	c.asset = asset
	c.cfg = cfg
	c.opts.profile = asset.Options["profile"]
	c.opts.scope = asset.Options["scope"]
	c.opts.options = asset.Options
	c.Filters = DiscoveryFiltersFromOpts(conf.Discover.GetFilter())
	return c, nil
}

func (c *AwsConnection) Hash() uint64 {
	// generate hash of the config options used to to initialize this connection,
	// we use this to avoid verifying a client with the same options more than once
	hash, err := hashstructure.Hash(c.opts, hashstructure.FormatV2, nil)
	if err != nil {
		log.Error().Err(err).Msg("unable to hash connection")
	}
	return hash
}

func (c *AwsConnection) Verify() (string, error) {
	identity, err := CheckIam(c.cfg.Copy())
	if err != nil {
		log.Debug().Err(err).Msg("could not gather details of AWS account")
		// try with govcloud region, store error to return it if this last option does not work
		err1 := err
		cfgCopy := c.cfg.Copy()
		cfgCopy.Region = "us-gov-west-1"
		identity, err = CheckIam(cfgCopy)
		if err != nil {
			return "", err1
		}
	}
	account := ""
	if identity.Account != nil {
		account = *identity.Account
	}

	return account, nil
}

func (c *AwsConnection) SetAccountId(id string) {
	if id != "" {
		c.accountId = id
	}
}

func (p *AwsConnection) AccountId() string {
	return p.accountId
}

func parseFlagsForConnectionOptions(m map[string]string, creds []*vault.Credential) []ConnectionOption {
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
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
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

func (p *AwsConnection) Profile() string {
	return p.opts.profile
}

func (p *AwsConnection) Scope() string {
	return p.opts.scope
}

func (p *AwsConnection) ConnectionOptions() map[string]string {
	return p.opts.options
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
	regionLimits := h.Filters.General.Regions
	if len(regionLimits) > 0 {
		log.Debug().Interface("regions", regionLimits).Msg("using region limits")
		// cache the regions as part of the provider instance
		h.clientcache.Store("_regions", &CacheEntry{Data: regionLimits})
		return regionLimits, nil
	}
	// if no cache, get regions using ec2 client (using the ssm list global regions does not give the same list)
	log.Debug().Msg("no region cache or region limits found. fetching regions")
	regions := []string{}
	svc := h.Ec2(h.cfg.Region)
	ctx := context.Background()

	// DescribeRegions works to get the list of enabled regions for the account ( each account of organization)
	// but this does not mean the respective service endpoint is available in that region. They will timeout instead of failing fast
	// (e.g. EKS,KMS,Sagemaker is for example not available in ap-southeast-1 etc)
	// This also does not cover SCPs that might block access to certain regions.
	res, err := svc.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		log.Warn().Err(err).Msg("unable to describe regions")
		// when we can't use `DescribeRegions` we will fallback to:
		// 1. Account list-regions
		// 2. Public regional table + region access verification
		enabledRegions, fallbackErr := h.fallbackGetEnabledRegions(ctx)
		if fallbackErr != nil {
			log.Warn().Err(fallbackErr).Msg("unable to list regions from fallback options")
			return regions, err
		}
		regions = enabledRegions
	} else {
		for _, region := range res.Regions {
			regions = append(regions, *region.RegionName)
		}
	}

	// ensure excluded regions are discarded
	filteredRegions := []string{}
	for _, region := range regions {
		if !slices.Contains(h.Filters.General.ExcludeRegions, region) {
			filteredRegions = append(filteredRegions, region)
		}
	}

	if len(filteredRegions) != len(regions) {
		log.Debug().
			Strs("filtered_regions", filteredRegions).
			Msg("list of regions changed based of applied filters")
	}

	// cache the regions as part of the provider instance
	h.clientcache.Store("_regions", &CacheEntry{Data: filteredRegions})
	return filteredRegions, nil
}

// fallbackGetEnabledRegions tries multiple ways to return the list of enabled regions.
//
// NOTE use this only if `DescribeRegions` doesn't work
func (h *AwsConnection) fallbackGetEnabledRegions(ctx context.Context) (regions []string, err error) {
	// 1. Account list-regions
	response, err := h.Account("").ListRegions(ctx, &account.ListRegionsInput{
		RegionOptStatusContains: []types.RegionOptStatus{
			types.RegionOptStatusEnabled,
			types.RegionOptStatusEnabling,
			types.RegionOptStatusEnabledByDefault,
		},
	})
	if err == nil {
		for _, region := range response.Regions {
			regions = append(regions, *region.RegionName)
		}
		log.Debug().Strs("regions", regions).Msg("regions>fallback> using account list-regions")
		return
	}

	log.Warn().Err(err).Msg("unable to list account regions")

	// 2. Public regional table + region access verification
	regionsFromTable, err := getRegionsFromRegionalTable()
	if err != nil {
		return
	}

	// verify which regions are enabled
	for _, region := range regionsFromTable {
		if h.isRegionEnabled(ctx, region) {
			regions = append(regions, region)
		}
	}

	log.Debug().Strs("regions", regions).Msg("using public regional table")
	return
}

// isRegionEnabled returns true if the provided region is enabled. We verify if a region is
// enabled by doing a simple request to that region.
func (h *AwsConnection) isRegionEnabled(ctx context.Context, region string) bool {
	_, err := h.STS(region).GetCallerIdentity(ctx, nil)
	return err == nil
}

type regionalTable struct {
	Metadata struct {
		Copyright     string `json:"copyright"`
		Disclaimer    string `json:"disclaimer"`
		FormatVersion string `json:"format:version"`
		SourceVersion string `json:"source:version"`
	} `json:"metadata"`
	Prices []struct {
		Attributes struct {
			AwsRegion      string `json:"aws:region"`
			AwsServiceName string `json:"aws:serviceName"`
			AwsServiceURL  string `json:"aws:serviceUrl"`
		} `json:"attributes"`
		ID string `json:"id"`
	} `json:"prices"`
}

// getRegionsFromRegionalTable is a workaround for cases where the DescribeRegions API
// is blocked. This function returns all possible AWS regions using a well known regional
// table provided by AWS.
//
// https://api.regional-table.region-services.aws.a2z.com/index.json
//
// NOTE: if we need to validate that we have access to that region or, that the region is
// enabled, we can improve this function to do STS identity calls for all regions.
func getRegionsFromRegionalTable() (regions []string, err error) {
	resp, err := http.Get("https://api.regional-table.region-services.aws.a2z.com/index.json")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var regionalTableJSON regionalTable
	err = json.Unmarshal(body, &regionalTableJSON)
	if err != nil {
		return
	}

	for _, p := range regionalTableJSON.Prices {
		if p.Attributes.AwsRegion != "" {
			regions = append(regions, p.Attributes.AwsRegion)
		}
	}
	slices.Sort(regions)
	regions = slices.Compact(regions)
	return
}
