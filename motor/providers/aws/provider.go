package aws

import (
	"context"
	"sync"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/providers"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

type ProviderOption func(charp *Provider)

func WithEndpoint(apiEndpoint string) ProviderOption {
	return func(p *Provider) {
		localResolverFn := func(service, region string) (aws_sdk.Endpoint, error) {
			return aws_sdk.Endpoint{
				URL:               apiEndpoint,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		}
		p.awsConfigOptions = append(p.awsConfigOptions, config.WithEndpointResolver(aws_sdk.EndpointResolverFunc(localResolverFn)))
	}
}

func WithRegion(region string) ProviderOption {
	return func(p *Provider) {
		p.awsConfigOptions = append(p.awsConfigOptions, config.WithRegion(region))
	}
}

func WithProfile(profile string) ProviderOption {
	return func(p *Provider) {
		p.awsConfigOptions = append(p.awsConfigOptions, config.WithSharedConfigProfile(profile))
	}
}

func TransportOptions(opts map[string]string) []ProviderOption {
	// extract config options
	transportOpts := []ProviderOption{}
	if apiEndpoint, ok := opts["endpoint-url"]; ok {
		transportOpts = append(transportOpts, WithEndpoint(apiEndpoint))
	}

	if awsRegion, ok := opts["region"]; ok {
		transportOpts = append(transportOpts, WithRegion(awsRegion))
	}

	if awsProfile, ok := opts["profile"]; ok {
		transportOpts = append(transportOpts, WithProfile(awsProfile))
	}
	return transportOpts
}

func New(pCfg *providers.Config, opts ...ProviderOption) (*Provider, error) {
	if pCfg.Backend != providers.ProviderType_AWS {
		return nil, providers.ErrProviderTypeDoesNotMatch
	}

	t := &Provider{
		awsConfigOptions: []func(*config.LoadOptions) error{},
	}

	for _, opt := range opts {
		opt(t)
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), t.awsConfigOptions...)
	if err != nil {
		return nil, errors.Wrap(err, "could not load aws configuration")
	}

	t.config = cfg

	// gather information about the aws account
	identity, err := CheckIam(t.config)
	if err != nil {
		log.Warn().Err(err).Msg("could not gather details of AWS account")
		// do not error since this break with localstack
	} else {
		t.info = Info{
			Account: toString(identity.Account),
			Arn:     toString(identity.Arn),
			UserId:  toString(identity.UserId),
		}
	}

	return t, nil
}

func toString(i *string) string {
	if i == nil {
		return ""
	}
	return *i
}

type Info struct {
	Account string
	Arn     string
	UserId  string
}

type Provider struct {
	config             aws_sdk.Config
	awsConfigOptions   []func(*config.LoadOptions) error
	selectedPlatformID string
	info               Info
	cache              Cache
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_AWS,
	}
}

func (p *Provider) Config() aws_sdk.Config {
	return p.config
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	return providers.RUNTIME_AWS
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (p *Provider) DefaultRegion() string {
	return p.config.Region
}

func (p *Provider) GetRegions() ([]string, error) {
	// check cache for regions list, return if exists
	c, ok := p.cache.Load("_regions")
	if ok {
		log.Debug().Msg("use regions from cache")
		return c.Data.([]string), nil
	}
	log.Debug().Msg("no region cache found. fetching regions")

	// if no cache, get regions using ec2 client (using the ssm list global regions does not give the same list)
	regions := []string{}
	svc := p.Ec2("us-east-1")
	ctx := context.Background()

	res, err := svc.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return regions, nil
	}
	for _, region := range res.Regions {
		regions = append(regions, *region.RegionName)
	}
	// cache the regions as part of the provider instance
	p.cache.Store("_regions", &CacheEntry{Data: regions})
	return regions, nil
}

// CacheEntry contains cached clients
type CacheEntry struct {
	Timestamp int64
	Valid     bool
	Data      interface{}
	Error     error
}

// Cache is a map containing CacheEntry values
type Cache struct{ sync.Map }

// Store a Cache Entry
func (c *Cache) Store(key string, v *CacheEntry) { c.Map.Store(key, v) }

// Load a Cache Entry
func (c *Cache) Load(key string) (*CacheEntry, bool) {
	res, ok := c.Map.Load(key)
	if res == nil {
		return nil, ok
	}
	return res.(*CacheEntry), ok
}

// Delete a Cache Entry
func (c *Cache) Delete(key string) { c.Map.Delete(key) }
