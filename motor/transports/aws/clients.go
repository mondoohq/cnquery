package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
)

func (t *Transport) Ec2(region string) *ec2.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_ec2_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached client")
		return c.Data.(*ec2.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := ec2.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Iam(region string) *iam.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_iam_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached client")
		return c.Data.(*iam.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := iam.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) S3(region string) *s3.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_s3_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached client")
		return c.Data.(*s3.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := s3.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Cloudtrail(region string) *cloudtrail.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_cloudtrail_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached client")
		return c.Data.(*cloudtrail.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := cloudtrail.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) ConfigService(region string) *configservice.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_config_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached client")
		return c.Data.(*configservice.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := configservice.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}
