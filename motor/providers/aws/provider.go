package aws

import (
	"context"
	"sync"

	"errors"
	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
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
		log.Debug().Str("region", awsRegion).Msg("using region")
		transportOpts = append(transportOpts, WithRegion(awsRegion))
	}

	if awsProfile, ok := opts["profile"]; ok {
		log.Debug().Str("profile", awsProfile).Msg("using aws profile")
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
		return nil, errors.Join(err, errors.New("could not load aws configuration"))
	}
	if cfg.Region == "" {
		log.Info().Msg("no AWS region found, using us-east-1")
		cfg.Region = "us-east-1" // in case the user has no region set, default to us-east-1
	}

	t.config = cfg
	var override string
	if pCfg.Options != nil {
		override = pCfg.Options["platform-override"]
	}
	// gather information about the aws account
	identity, err := CheckIam(t.config)
	if err != nil {
		log.Debug().Err(err).Msg("could not gather details of AWS account")
		// try with govcloud region
		t.config.Region = "us-gov-west-1"
		identity, err = CheckIam(t.config)
		if err != nil {
			log.Debug().Err(err).Msg("could not gather details of AWS account")
			return nil, err
		}
	}

	// either the regular check or the gov-check worked
	if err == nil {
		t.info = Info{
			Account:          toString(identity.Account),
			Arn:              toString(identity.Arn),
			UserId:           toString(identity.UserId),
			PlatformOverride: override,
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
	Account          string
	Arn              string
	UserId           string
	PlatformOverride string
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

func (p *Provider) PlatformInfo() *platform.Platform {
	return getPlatformForObject(p.info.PlatformOverride)
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
		// try with govcloud region
		svc := p.Ec2("us-gov-west-1")
		res, err = svc.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
		if err != nil {
			return regions, err
		}
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

func getPlatformForObject(platformName string) *platform.Platform {
	if platformName != "aws" && platformName != "" {
		return &platform.Platform{
			Name:    platformName,
			Title:   getTitleForPlatformName(platformName),
			Kind:    providers.Kind_KIND_AWS_OBJECT,
			Runtime: providers.RUNTIME_AWS,
		}
	}
	return &platform.Platform{
		Name:    "aws",
		Title:   "Amazon Web Services",
		Kind:    providers.Kind_KIND_API,
		Runtime: providers.RUNTIME_AWS,
	}
}

func getTitleForPlatformName(name string) string {
	switch name {
	case "aws-s3-bucket":
		return "AWS S3 Bucket"
	case "aws-cloudtrail-trail":
		return "AWS Cloudtrail Trail"
	case "aws-rds-dbinstance":
		return "AWS RDS DBInstance"
	case "aws-dynamodb-table":
		return "AWS DynamoDB Table"
	case "aws-redshift-cluster":
		return "AWS Redshift Cluster"
	case "aws-vpc":
		return "AWS VPC"
	case "aws-security-group":
		return "AWS Security Group"
	case "aws-ec2-volume":
		return "AWS EC2 Volume"
	case "aws-ec2-snapshot":
		return "AWS EC2 Snapshot"
	case "aws-iam-user":
		return "AWS IAM User"
	case "aws-iam-group":
		return "AWS IAM Group"
	case "aws-cloudwatch-loggroup":
		return "AWS Cloudwatch Loggroup"
	case "aws-lambda-function":
		return "AWS Lambda Function"
	case "aws-ecs-container":
		return "AWS ECS Container"
	case "aws-efs-filesystem":
		return "AWS EFS Filesystem"
	case "aws-gateway-restapi":
		return "AWS Gateway RESTAPI"
	case "aws-elb-loadbalancer":
		return "AWS ELB Load Balancer"
	case "aws-es-domain":
		return "AWS ES Domain"
	case "aws-kms-key":
		return "AWS KMS Key"
	case "aws-sagemaker-notebookinstance":
		return "AWS Sagemaker Notebook Instance"
	case "aws-ec2-instance":
		return "AWS EC2 Instance"
	case "aws-ssm-instance":
		return "AWS SSM Instance"
	case "aws-ecr-image":
		return "AWS ECR Image"
	}
	return "Amazon Web Services"
}
