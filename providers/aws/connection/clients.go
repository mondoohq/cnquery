package connection

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
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func (p *AwsConnection) Clients(name string, region string) interface{} {
	switch name {
	case "ec2":
		c := p.Ec2(region)
		p.clients["ec2"][region] = c
		return c
	}
	return nil
}

func (p *AwsConnection) Ec2(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := ec2.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Ecs(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := ecs.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Iam(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := iam.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Ecr(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := ecr.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) EcrPublic(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := ecrpublic.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) S3(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := s3.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) S3Control(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := s3control.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Cloudtrail(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := cloudtrail.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Cloudfront(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := cloudfront.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) ConfigService(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := configservice.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Kms(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := kms.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) CloudwatchLogs(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := cloudwatchlogs.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Cloudwatch(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := cloudwatch.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Sns(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := sns.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Ssm(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := ssm.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Efs(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := efs.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Apigateway(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := apigateway.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) ApplicationAutoscaling(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := applicationautoscaling.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Lambda(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := lambda.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Dynamodb(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := dynamodb.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Dms(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := databasemigrationservice.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Rds(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := rds.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Elasticache(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := elasticache.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Redshift(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := redshift.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) AccessAnalyzer(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := accessanalyzer.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Acm(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := acm.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Elb(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := elasticloadbalancing.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Elbv2(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := elasticloadbalancingv2.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Es(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := elasticsearchservice.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Sagemaker(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := sagemaker.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Autoscaling(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := autoscaling.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Backup(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := backup.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Codebuild(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := codebuild.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Emr(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := emr.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Guardduty(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := guardduty.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Secretsmanager(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := secretsmanager.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) SecurityHub(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := securityhub.NewFromConfig(cfg)

	return client
}

func (p *AwsConnection) Eks(region string) *eks.Client {
	// if no region value is sent in, use the configured region
	if len(region) == 0 {
		region = p.cfg.Region
	}

	// create the client
	cfg := p.cfg.Copy()
	cfg.Region = region
	client := eks.NewFromConfig(cfg)

	return client
}
