// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	"github.com/aws/aws-sdk-go-v2/service/account"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go-v2/service/appstream"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go-v2/service/dax"
	"github.com/aws/aws-sdk-go-v2/service/directoryservice"
	"github.com/aws/aws-sdk-go-v2/service/docdb"
	"github.com/aws/aws-sdk-go-v2/service/drs"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go-v2/service/emr"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/aws/aws-sdk-go-v2/service/fsx"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/inspector2"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
	"github.com/aws/aws-sdk-go-v2/service/macie2"
	"github.com/aws/aws-sdk-go-v2/service/memorydb"
	"github.com/aws/aws-sdk-go-v2/service/mq"
	"github.com/aws/aws-sdk-go-v2/service/neptune"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/pipes"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53domains"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/aws/aws-sdk-go-v2/service/shield"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/timestreaminfluxdb"
	"github.com/aws/aws-sdk-go-v2/service/timestreamwrite"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/aws/aws-sdk-go-v2/service/workdocs"
	"github.com/aws/aws-sdk-go-v2/service/workspaces"
	"github.com/aws/aws-sdk-go-v2/service/workspacesweb"
	"github.com/rs/zerolog/log"
)

// CacheEntry contains cached clients
type CacheEntry struct {
	Timestamp int64
	Valid     bool
	Data      any
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

func (t *AwsConnection) Wafv2(region string) *wafv2.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_wafv2_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached wafv2 client")
		return c.Data.(*wafv2.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := wafv2.NewFromConfig(cfg)

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

func (t *AwsConnection) Inspector(region string) *inspector2.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_inspector_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached inspector client")
		return c.Data.(*inspector2.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := inspector2.NewFromConfig(cfg)

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

func (t *AwsConnection) Sqs(region string) *sqs.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_sqs_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached sqs client")
		return c.Data.(*sqs.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := sqs.NewFromConfig(cfg)

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
		log.Debug().Msg("use cached efs client")
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

func (t *AwsConnection) EventBridge(region string) *eventbridge.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_eventbridge_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached eventbridge client")
		return c.Data.(*eventbridge.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := eventbridge.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Fsx(region string) *fsx.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_fsx_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached fsx client")
		return c.Data.(*fsx.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := fsx.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Firehose(region string) *firehose.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_firehose_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached firehose client")
		return c.Data.(*firehose.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := firehose.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Kinesis(region string) *kinesis.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_kinesis_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached kinesis client")
		return c.Data.(*kinesis.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := kinesis.NewFromConfig(cfg)

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

func (t *AwsConnection) Appstream(region string) *appstream.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_appstream_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached appstream client")
		return c.Data.(*appstream.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := appstream.NewFromConfig(cfg)

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

func (t *AwsConnection) Macie2(region string) *macie2.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_macie2_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached macie2 client")
		return c.Data.(*macie2.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := macie2.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Memorydb(region string) *memorydb.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_memorydb_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached memorydb client")
		return c.Data.(*memorydb.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := memorydb.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Kafka(region string) *kafka.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_kafka_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached kafka client")
		return c.Data.(*kafka.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := kafka.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Mq(region string) *mq.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_mq_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached mq client")
		return c.Data.(*mq.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := mq.NewFromConfig(cfg)

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

func (t *AwsConnection) Dax(region string) *dax.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_dax_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached dax client")
		return c.Data.(*dax.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := dax.NewFromConfig(cfg)

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

func (t *AwsConnection) Route53(region string) *route53.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_route53_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached route53 client")
		return c.Data.(*route53.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := route53.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Neptune(region string) *neptune.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_neptune_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached neptune client")
		return c.Data.(*neptune.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region

	// Create a Neptune client from just a session.
	client := neptune.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) OpenSearch(region string) *opensearch.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_opensearch_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached opensearch client")
		return c.Data.(*opensearch.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := opensearch.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

// TimestreamLiveAnalytics returns a Timestream client for Live Analytics
func (t *AwsConnection) TimestreamLiveAnalytics(region string) *timestreamwrite.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_timestream_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached timestreamwrite client")
		return c.Data.(*timestreamwrite.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region

	// Create a Neptune client from just a session.
	client := timestreamwrite.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) TimestreamInfluxDB(region string) *timestreaminfluxdb.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_timestream_influxdb_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached timestreaminfluxdb client")
		return c.Data.(*timestreaminfluxdb.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := timestreaminfluxdb.NewFromConfig(cfg)

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

func (t *AwsConnection) Athena(region string) *athena.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_athena_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached athena client")
		return c.Data.(*athena.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := athena.NewFromConfig(cfg)

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

func (t *AwsConnection) Drs(region string) *drs.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_drs_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached drs client")
		return c.Data.(*drs.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := drs.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) DirectoryService(region string) *directoryservice.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_directoryservice_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached directoryservice client")
		return c.Data.(*directoryservice.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := directoryservice.NewFromConfig(cfg)

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

func (t *AwsConnection) CodeDeploy(region string) *codedeploy.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_codedeploy" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached codebuild client")
		return c.Data.(*codedeploy.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := codedeploy.NewFromConfig(cfg)

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

func (t *AwsConnection) Shield(region string) *shield.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_shield_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached shield client")
		return c.Data.(*shield.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := shield.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) NetworkFirewall(region string) *networkfirewall.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_networkfirewall_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached networkfirewall client")
		return c.Data.(*networkfirewall.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := networkfirewall.NewFromConfig(cfg)

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

func (t *AwsConnection) Account(region string) *account.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_account_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached account client")
		return c.Data.(*account.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := account.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) WorkDocs(region string) *workdocs.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_workdocs_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached workdocs client")
		return c.Data.(*workdocs.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := workdocs.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Workspaces(region string) *workspaces.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_workspaces_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached workspaces client")
		return c.Data.(*workspaces.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := workspaces.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) WorkspacesWeb(region string) *workspacesweb.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_workspacesweb_" + region
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached workspacesweb client")
		return c.Data.(*workspacesweb.Client)
	}
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := workspacesweb.NewFromConfig(cfg)
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) STS(region string) *sts.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_sts_" + region

	// check for cached client and return it if it exists
	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached sts client")
		return c.Data.(*sts.Client)
	}

	// create the client
	cfg := t.cfg.Copy()
	cfg.Region = region
	client := sts.NewFromConfig(cfg)

	// cache it
	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Glue(region string) *glue.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_glue_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached glue client")
		return c.Data.(*glue.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := glue.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Route53Domains(region string) *route53domains.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_route53domains_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached route53domains client")
		return c.Data.(*route53domains.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := route53domains.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) CognitoIdentity(region string) *cognitoidentity.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_cognitoidentity_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached cognitoidentity client")
		return c.Data.(*cognitoidentity.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := cognitoidentity.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) CognitoIdentityProvider(region string) *cognitoidentityprovider.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_cognitoidentityprovider_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached cognitoidentityprovider client")
		return c.Data.(*cognitoidentityprovider.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := cognitoidentityprovider.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) DocumentDB(region string) *docdb.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_docdb_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached docdb client")
		return c.Data.(*docdb.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := docdb.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) ElasticBeanstalk(region string) *elasticbeanstalk.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_elasticbeanstalk_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached elasticbeanstalk client")
		return c.Data.(*elasticbeanstalk.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := elasticbeanstalk.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Batch(region string) *batch.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_batch_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached batch client")
		return c.Data.(*batch.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := batch.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) CloudFormation(region string) *cloudformation.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_cloudformation_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached cloudformation client")
		return c.Data.(*cloudformation.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := cloudformation.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Lightsail(region string) *lightsail.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_lightsail_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached lightsail client")
		return c.Data.(*lightsail.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := lightsail.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Pipes(region string) *pipes.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_pipes_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached pipes client")
		return c.Data.(*pipes.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := pipes.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}

func (t *AwsConnection) Scheduler(region string) *scheduler.Client {
	if len(region) == 0 {
		region = t.cfg.Region
	}
	cacheVal := "_scheduler_" + region

	c, ok := t.clientcache.Load(cacheVal)
	if ok {
		log.Debug().Msg("use cached scheduler client")
		return c.Data.(*scheduler.Client)
	}

	cfg := t.cfg.Copy()
	cfg.Region = region
	client := scheduler.NewFromConfig(cfg)

	t.clientcache.Store(cacheVal, &CacheEntry{Data: client})
	return client
}
