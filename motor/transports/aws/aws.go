package aws

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

func New(tc *transports.TransportConfig) (*Transport, error) {
	if tc.Backend != transports.TransportBackend_CONNECTION_AWS {
		return nil, errors.New("backend is not supported for aws transport")
	}

	configs := []external.Config{}
	if tc.Options != nil && len(tc.Options["profile"]) > 0 {
		configs = append(configs, external.WithSharedConfigProfile(tc.Options["profile"]))
	}

	cfg, err := external.LoadDefaultAWSConfig(configs...)
	if err != nil {
		return nil, errors.Wrap(err, "could not load aws configuration")
	}

	if tc.Options != nil && len(tc.Options["region"]) > 0 {
		cfg.Region = tc.Options["region"]
	}

	identity, err := CheckIam(cfg)
	if err != nil {
		return nil, err
	}

	return &Transport{
		config: cfg,
		opts:   tc.Options,
		info: Info{
			Account: toString(identity.Account),
			Arn:     toString(identity.Arn),
			UserId:  toString(identity.UserId),
		},
	}, nil
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
	opts               map[string]string
	selectedPlatformID string
	info               Info
	cache              Cache
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("aws does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("aws does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_AWS,
	}
}

func (t *Transport) Config() aws_sdk.Config {
	return t.config
}

func (t *Transport) Options() map[string]string {
	return t.opts
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return transports.RUNTIME_AWS
}

func (t *Transport) GetRegions() []string {
	// check cache for regions list, return if exists
	c, ok := t.cache.Load("_regions")
	if ok {
		log.Info().Msg("use regions from cache")
		return c.Data.([]string)
	}
	log.Info().Msg("no region cache found. fetching regions")

	// if no cache, get regions using ec2 client (using the ssm list global regions does not give the same list)
	regions := []string{}
	svc := t.Ec2("us-east-1")
	ctx := context.Background()

	res, err := svc.DescribeRegionsRequest(&ec2.DescribeRegionsInput{}).Send(ctx)
	if err != nil {
		log.Err(err)
	}

	for _, region := range res.Regions {
		regions = append(regions, *region.RegionName)
	}
	// cache the regions as part of the transport object
	t.cache.Store("_regions", &CacheEntry{Data: regions})
	return regions
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
