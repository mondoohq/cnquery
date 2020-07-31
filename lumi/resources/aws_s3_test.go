package resources_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestCloudtrail(t *testing.T) {

	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile("mondoo-inc"))
	if err != nil {
		panic(err)
	}
	cfg.Region = endpoints.UsEast1RegionID

	// iterate over each region?
	svc := s3.New(cfg)
	require.NoError(t, err)
	ctx := context.Background()

	res, err := svc.ListBucketsRequest(&s3.ListBucketsInput{}).Send(ctx)

	svc.GetBucketTaggingRequest(&s3.GetBucketTaggingInput{}).Send(ctx)
	svc.GetBucketLocationRequest(&s3.GetBucketLocationInput{}).Send(ctx)
	svc.GetBucketAclRequest(&s3.GetBucketAclInput{}).Send(ctx)
	svc.GetBucketEncryptionRequest(&s3.GetBucketEncryptionInput{}).Send(ctx)
	svc.GetBucketLocationRequest(&s3.GetBucketLocationInput{}).Send(ctx)
	svc.GetBucketPolicyRequest(&s3.GetBucketPolicyInput{}).Send(ctx)
	svc.GetBucketCorsRequest(&s3.GetBucketCorsInput{}).Send(ctx)
	svc.GetBucketMetricsConfigurationRequest(&s3.GetBucketMetricsConfigurationInput{}).Send(ctx)
}
