package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"

	"github.com/aws/aws-sdk-go-v2/service/ecs"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go-v2/service/emr"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/rs/zerolog/log"
)

func (t *Provider) Ec2(region string) *ec2.Client {
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
	client := ec2.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Ecs(region string) *ecs.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_ecs_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached ecs client")
		return c.Data.(*ecs.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := ecs.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Iam(region string) *iam.Client {
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
	client := iam.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Ecr(region string) *ecr.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_ecr_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached ecr client")
		return c.Data.(*ecr.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := ecr.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) EcrPublic(region string) *ecrpublic.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_ecrpublic_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached ecrpublic client")
		return c.Data.(*ecrpublic.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := ecrpublic.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) S3(region string) *s3.Client {
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
	client := s3.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) S3Control(region string) *s3control.Client {
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
	client := s3control.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Cloudtrail(region string) *cloudtrail.Client {
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
	client := cloudtrail.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Cloudfront(region string) *cloudfront.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_cloudfront_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached cloudfront client")
		return c.Data.(*cloudfront.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := cloudfront.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) ConfigService(region string) *configservice.Client {
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
	client := configservice.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Kms(region string) *kms.Client {
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
	client := kms.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) CloudwatchLogs(region string) *cloudwatchlogs.Client {
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
	client := cloudwatchlogs.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Cloudwatch(region string) *cloudwatch.Client {
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
	client := cloudwatch.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Sns(region string) *sns.Client {
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
	client := sns.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Ssm(region string) *ssm.Client {
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
	client := ssm.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Efs(region string) *efs.Client {
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
	client := efs.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Apigateway(region string) *apigateway.Client {
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
	client := apigateway.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) ApplicationAutoscaling(region string) *applicationautoscaling.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_applicationautoscaling_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached applicationautoscaling client")
		return c.Data.(*applicationautoscaling.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := applicationautoscaling.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Lambda(region string) *lambda.Client {
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
	client := lambda.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Dynamodb(region string) *dynamodb.Client {
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
	client := dynamodb.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Dms(region string) *databasemigrationservice.Client {
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
	client := databasemigrationservice.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Rds(region string) *rds.Client {
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
	client := rds.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Elasticache(region string) *elasticache.Client {
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
	client := elasticache.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Redshift(region string) *redshift.Client {
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
	client := redshift.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) AccessAnalyzer(region string) *accessanalyzer.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_accessanalyzer_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached access analyzer client")
		return c.Data.(*accessanalyzer.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := accessanalyzer.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Acm(region string) *acm.Client {
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
	client := acm.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Elb(region string) *elasticloadbalancing.Client {
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
	client := elasticloadbalancing.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Elbv2(region string) *elasticloadbalancingv2.Client {
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
	client := elasticloadbalancingv2.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Es(region string) *elasticsearchservice.Client {
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
	client := elasticsearchservice.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Sagemaker(region string) *sagemaker.Client {
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
	client := sagemaker.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Autoscaling(region string) *autoscaling.Client {
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
	client := autoscaling.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Backup(region string) *backup.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_backup_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached backup client")
		return c.Data.(*backup.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := backup.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Codebuild(region string) *codebuild.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_codebuild_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached codebuild client")
		return c.Data.(*codebuild.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := codebuild.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Emr(region string) *emr.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_emr_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached emr client")
		return c.Data.(*emr.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := emr.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Guardduty(region string) *guardduty.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_guardduty_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached guardduty client")
		return c.Data.(*guardduty.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := guardduty.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Secretsmanager(region string) *secretsmanager.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_secretsmanager_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached secretsmanager client")
		return c.Data.(*secretsmanager.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := secretsmanager.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Securityhub(region string) *securityhub.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_securityhub_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached securityhub client")
		return c.Data.(*securityhub.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := securityhub.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *Provider) Eks(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.config.Region
	}
	cacheVal := "_eks_" + region

	// check for cached client and return it if it exists
	c, ok := t.cache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached eks client")
		return c.Data.(*eks.Client)
	}

	// create the client
	cfg := t.config.Copy()
	cfg.Region = region
	client := eks.NewFromConfig(cfg)

	// cache it
	t.cache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}
