package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
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

func (t *Transport) Dms(region string) *databasemigrationservice.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_dms_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached dms client")
		return c.Data.(*databasemigrationservice.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := databasemigrationservice.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Rds(region string) *rds.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_rds_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached rds client")
		return c.Data.(*rds.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := rds.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Elasticache(region string) *elasticache.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_elasticache_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached elasticache client")
		return c.Data.(*elasticache.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := elasticache.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Redshift(region string) *redshift.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_redshift_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached redshift client")
		return c.Data.(*redshift.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := redshift.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Acm(region string) *acm.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_acm_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached acm client")
		return c.Data.(*acm.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := acm.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Elb(region string) *elasticloadbalancing.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_elb_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached elb client")
		return c.Data.(*elasticloadbalancing.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := elasticloadbalancing.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Elbv2(region string) *elasticloadbalancingv2.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_elbv2_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached elbv2 client")
		return c.Data.(*elasticloadbalancingv2.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := elasticloadbalancingv2.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Es(region string) *elasticsearchservice.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_es_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached es client")
		return c.Data.(*elasticsearchservice.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := elasticsearchservice.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Sagemaker(region string) *sagemaker.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_sagemaker_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached sagemaker client")
		return c.Data.(*sagemaker.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := sagemaker.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Transport) Autoscaling(region string) *autoscaling.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_autoscaling_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached autoscaling client")
		return c.Data.(*autoscaling.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := autoscaling.New(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}
