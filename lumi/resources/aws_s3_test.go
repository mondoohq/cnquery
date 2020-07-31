package resources_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestS3(t *testing.T) {

	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile("mondoo-inc"))
	if err != nil {
		panic(err)
	}
	cfg.Region = endpoints.UsEast1RegionID

	// iterate over each region?
	svc := s3.New(cfg)
	require.NoError(t, err)
	ctx := context.Background()

	// res, err := svc.ListBucketsRequest(&s3.ListBucketsInput{}).Send(ctx)
	// require.NoError(t, err)
	// fmt.Printf("%v", res)

	bucket := "multi-reagion-trail-test"
	// // policy
	// policy, err := svc.GetBucketPolicyRequest(&s3.GetBucketPolicyInput{
	// 	Bucket: &bucket,
	// }).Send(ctx)
	// require.NoError(t, err)
	// fmt.Printf("%v", policy)

	// // tags
	// tags, err := svc.GetBucketTaggingRequest(&s3.GetBucketTaggingInput{
	// 	Bucket: &bucket,
	// }).Send(ctx)
	// // aws code NoSuchTagSetError in case no tag is set
	// // require.NoError(t, err)
	// fmt.Printf("%v", tags)

	// cors
	// cors, err := svc.GetBucketCorsRequest(&s3.GetBucketCorsInput{
	// 	Bucket: &bucket,
	// }).Send(ctx)
	// // require.NoError(t, err)
	// // if a bucket has no cors configuration, we get "NoSuchCORSConfiguration"
	// fmt.Printf("%v", cors)

	// acl
	acl, err := svc.GetBucketAclRequest(&s3.GetBucketAclInput{
		Bucket: &bucket,
	}).Send(ctx)
	// require.NoError(t, err)
	fmt.Printf("%v", acl)

	// encryption, err := svc.GetBucketEncryptionRequest(&s3.GetBucketEncryptionInput{
	// 	Bucket: &bucket,
	// }).Send(ctx)
	// // require.NoError(t, err), returns "ServerSideEncryptionConfigurationNotFoundError" if not set
	// fmt.Printf("%v", encryption)

	location, err := svc.GetBucketLocationRequest(&s3.GetBucketLocationInput{
		Bucket: &bucket,
	}).Send(ctx)
	// require.NoError(t, err)
	// returns the region of the location
	// us-east-1 returns ""
	// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketLocation.html#API_GetBucketLocation_ResponseSyntax
	fmt.Printf("%v", location)

	metrics, err := svc.GetBucketMetricsConfigurationRequest(&s3.GetBucketMetricsConfigurationInput{
		Bucket: &bucket,
	}).Send(ctx)
	// require.NoError(t, err)
	fmt.Printf("%v", metrics)
}
