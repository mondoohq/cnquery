package aws

import (
	"context"
	"sync"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/fsutil"
)

var (
	_ providers.Transport                   = (*Transport)(nil)
	_ providers.TransportPlatformIdentifier = (*Transport)(nil)
)

type TransportOption func(chart *Transport)

func WithEndpoint(apiEndpoint string) TransportOption {
	return func(t *Transport) {
		localResolverFn := func(service, region string) (aws_sdk.Endpoint, error) {
			return aws_sdk.Endpoint{
				URL:               apiEndpoint,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		}
		t.awsConfigOptions = append(t.awsConfigOptions, config.WithEndpointResolver(aws_sdk.EndpointResolverFunc(localResolverFn)))
	}
}

func WithRegion(region string) TransportOption {
	return func(t *Transport) {
		t.awsConfigOptions = append(t.awsConfigOptions, config.WithRegion(region))
	}
}

func WithProfile(profile string) TransportOption {
	return func(t *Transport) {
		t.awsConfigOptions = append(t.awsConfigOptions, config.WithSharedConfigProfile(profile))
	}
}

func TransportOptions(opts map[string]string) []TransportOption {
	// extract config options
	transportOpts := []TransportOption{}
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

func New(tc *providers.TransportConfig, opts ...TransportOption) (*Transport, error) {
	if tc.Backend != providers.TransportBackend_CONNECTION_AWS {
		return nil, errors.New("backend is not supported for aws transport")
	}

	t := &Transport{
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

type Transport struct {
	config             aws_sdk.Config
	awsConfigOptions   []func(*config.LoadOptions) error
	selectedPlatformID string
	info               Info
	cache              Cache
}

func (t *Transport) RunCommand(command string) (*providers.Command, error) {
	return nil, errors.New("aws does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (providers.FileInfoDetails, error) {
	return providers.FileInfoDetails{}, errors.New("aws does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_AWS,
	}
}

func (t *Transport) Config() aws_sdk.Config {
	return t.config
}

func (t *Transport) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return providers.RUNTIME_AWS
}

func (t *Transport) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (t *Transport) DefaultRegion() string {
	return t.config.Region
}

func (t *Transport) GetRegions() ([]string, error) {
	// check cache for regions list, return if exists
	c, ok := t.cache.Load("_regions")
	if ok {
		log.Debug().Msg("use regions from cache")
		return c.Data.([]string), nil
	}
	log.Debug().Msg("no region cache found. fetching regions")

	// if no cache, get regions using ec2 client (using the ssm list global regions does not give the same list)
	regions := []string{}
	svc := t.Ec2("us-east-1")
	ctx := context.Background()

	res, err := svc.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return regions, nil
	}
	for _, region := range res.Regions {
		regions = append(regions, *region.RegionName)
	}
	// cache the regions as part of the transport object
	t.cache.Store("_regions", &CacheEntry{Data: regions})
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
