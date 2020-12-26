package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
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
		log.Debug().Msg("use cached ec2 client")
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
		log.Debug().Msg("use cached iam client")
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
		log.Debug().Msg("use cached s3 client")
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

func (t *Transport) S3Control(region string) *s3control.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_s3control_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached s3control client")
		return c.Data.(*s3control.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := s3control.New(cfg)

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
		log.Debug().Msg("use cached cloudtrail client")
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
		log.Debug().Msg("use cached config client")
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

func (t *Transport) Kms(region string) *kms.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_kms_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached kms client")
		return c.Data.(*kms.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := kms.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) CloudwatchLogs(region string) *cloudwatchlogs.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_cloudwatchlogs_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached cloudwatchlogs client")
		return c.Data.(*cloudwatchlogs.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := cloudwatchlogs.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Cloudwatch(region string) *cloudwatch.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_cloudwatch_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached cloudwatch client")
		return c.Data.(*cloudwatch.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := cloudwatch.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Sns(region string) *sns.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_sns_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached sns client")
		return c.Data.(*sns.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := sns.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Ssm(region string) *ssm.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_ssm_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached ssm client")
		return c.Data.(*ssm.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := ssm.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Efs(region string) *efs.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_efs_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached ssm client")
		return c.Data.(*efs.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := efs.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Apigateway(region string) *apigateway.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_apigateway_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached apigateway client")
		return c.Data.(*apigateway.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := apigateway.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Lambda(region string) *lambda.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_lambda_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached lambda client")
		return c.Data.(*lambda.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := lambda.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Dynamodb(region string) *dynamodb.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_dynamodb_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached dynamodb client")
		return c.Data.(*dynamodb.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := dynamodb.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}
