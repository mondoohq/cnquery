package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (t *Transport) Ec2() *ec2.Client {
	return ec2.New(t.config)
}

func (t *Transport) Iam() *iam.Client {
	return iam.New(t.config)
}

func (t *Transport) S3(region string) *s3.Client {
	cfg := t.config.Copy()
	if region == "" {
		cfg.Region = endpoints.UsEast1RegionID
	} else {
		// NOTE: for s3 buckets, we need to switch the region to gather the policy documents
		cfg.Region = region
	}

	// iterate over each region?
	svc := s3.New(cfg)
	return svc
}

func (t *Transport) Cloudtrail() *cloudtrail.Client {
	return cloudtrail.New(t.config)
}

func (t *Transport) ConfigService() *configservice.Client {
	return configservice.New(t.config)
}

func (t *Transport) Organizations() *organizations.Client {
	return organizations.New(t.config)
}
