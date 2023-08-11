package connection

import (
	"sync"

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
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go-v2/service/emr"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
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

// CacheEntry contains cached clients
type CacheEntry struct {
	Timestamp int64
	Valid     bool
	Data      interface{}
	Error     error
}

// Cache is a map containing CacheEntry values
type ClientsCache struct{ sync.Map }

// Store a Cache Entry
func (c *ClientsCache) Store(key string, v *CacheEntry) { c.Map.Store(key, v) }

// Load a Cache Entry
func (c *ClientsCache) Load(key string) (*CacheEntry, bool) {
	res, ok := c.Map.Load(key)
	if res == nil {
		return nil, ok
	}
	return res.(*CacheEntry), ok
}

// Delete a Cache Entry
func (c *ClientsCache) Delete(key string) { c.Map.Delete(key) }

func (t *AwsConnection) Organizations(region string) *organizations.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_organizations_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached organizations client")
		return c.Data.(*organizations.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := organizations.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Ec2(region string) *ec2.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_ec2_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached ec2 client")
		return c.Data.(*ec2.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := ec2.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Ecs(region string) *ecs.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_ecs_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached ecs client")
		return c.Data.(*ecs.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := ecs.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Iam(region string) *iam.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_iam_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached iam client")
		return c.Data.(*iam.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := iam.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Ecr(region string) *ecr.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_ecr_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached ecr client")
		return c.Data.(*ecr.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := ecr.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) EcrPublic(region string) *ecrpublic.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_ecrpublic_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached ecrpublic client")
		return c.Data.(*ecrpublic.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := ecrpublic.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) S3(region string) *s3.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_s3_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached s3 client")
		return c.Data.(*s3.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := s3.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) S3Control(region string) *s3control.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_s3control_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached s3control client")
		return c.Data.(*s3control.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := s3control.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Cloudtrail(region string) *cloudtrail.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_cloudtrail_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached cloudtrail client")
		return c.Data.(*cloudtrail.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := cloudtrail.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Cloudfront(region string) *cloudfront.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_cloudfront_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached cloudfront client")
		return c.Data.(*cloudfront.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := cloudfront.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) ConfigService(region string) *configservice.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_config_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached config client")
		return c.Data.(*configservice.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := configservice.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Kms(region string) *kms.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_kms_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached kms client")
		return c.Data.(*kms.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := kms.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) CloudwatchLogs(region string) *cloudwatchlogs.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_cloudwatchlogs_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached cloudwatchlogs client")
		return c.Data.(*cloudwatchlogs.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := cloudwatchlogs.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Cloudwatch(region string) *cloudwatch.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_cloudwatch_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached cloudwatch client")
		return c.Data.(*cloudwatch.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := cloudwatch.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Sns(region string) *sns.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_sns_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached sns client")
		return c.Data.(*sns.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := sns.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Ssm(region string) *ssm.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_ssm_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached ssm client")
		return c.Data.(*ssm.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := ssm.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Efs(region string) *efs.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_efs_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached ssm client")
		return c.Data.(*efs.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := efs.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Apigateway(region string) *apigateway.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_apigateway_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached apigateway client")
		return c.Data.(*apigateway.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := apigateway.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) ApplicationAutoscaling(region string) *applicationautoscaling.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_applicationautoscaling_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached applicationautoscaling client")
		return c.Data.(*applicationautoscaling.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := applicationautoscaling.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Lambda(region string) *lambda.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_lambda_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached lambda client")
		return c.Data.(*lambda.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := lambda.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Dynamodb(region string) *dynamodb.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_dynamodb_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached dynamodb client")
		return c.Data.(*dynamodb.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := dynamodb.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Dms(region string) *databasemigrationservice.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_dms_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached dms client")
		return c.Data.(*databasemigrationservice.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := databasemigrationservice.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Rds(region string) *rds.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_rds_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached rds client")
		return c.Data.(*rds.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := rds.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Elasticache(region string) *elasticache.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_elasticache_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached elasticache client")
		return c.Data.(*elasticache.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := elasticache.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Redshift(region string) *redshift.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_redshift_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached redshift client")
		return c.Data.(*redshift.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := redshift.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) AccessAnalyzer(region string) *accessanalyzer.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_accessanalyzer_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached access analyzer client")
		return c.Data.(*accessanalyzer.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := accessanalyzer.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Acm(region string) *acm.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_acm_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached acm client")
		return c.Data.(*acm.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := acm.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Elb(region string) *elasticloadbalancing.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_elb_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached elb client")
		return c.Data.(*elasticloadbalancing.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := elasticloadbalancing.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Elbv2(region string) *elasticloadbalancingv2.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_elbv2_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached elbv2 client")
		return c.Data.(*elasticloadbalancingv2.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := elasticloadbalancingv2.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Es(region string) *elasticsearchservice.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_es_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached es client")
		return c.Data.(*elasticsearchservice.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := elasticsearchservice.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Sagemaker(region string) *sagemaker.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_sagemaker_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached sagemaker client")
		return c.Data.(*sagemaker.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := sagemaker.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Autoscaling(region string) *autoscaling.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_autoscaling_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached autoscaling client")
		return c.Data.(*autoscaling.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := autoscaling.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Backup(region string) *backup.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_backup_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached backup client")
		return c.Data.(*backup.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := backup.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Codebuild(region string) *codebuild.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_codebuild_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached codebuild client")
		return c.Data.(*codebuild.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := codebuild.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Emr(region string) *emr.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_emr_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached emr client")
		return c.Data.(*emr.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := emr.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Guardduty(region string) *guardduty.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_guardduty_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached guardduty client")
		return c.Data.(*guardduty.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := guardduty.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Secretsmanager(region string) *secretsmanager.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_secretsmanager_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached secretsmanager client")
		return c.Data.(*secretsmanager.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := secretsmanager.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Securityhub(region string) *securityhub.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_securityhub_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached securityhub client")
		return c.Data.(*securityhub.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := securityhub.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Eks(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_eks_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached eks client")
		return c.Data.(*eks.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := eks.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}
