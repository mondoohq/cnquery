package resources_test

// import (
// 	"context"
// 	"fmt"
// 	"net/url"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"

// 	"github.com/aws/aws-sdk-go-v2/aws/arn"
// 	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
// 	"github.com/aws/aws-sdk-go-v2/aws/external"
// 	"github.com/aws/aws-sdk-go-v2/service/iam"
// )

// func TestResource_AwsIamCredentialReport(t *testing.T) {
// 	t.Run("run a aws iam credential report", func(t *testing.T) {
// 		res := testQuery(t, "aws.iam.credentialreport.length")
// 		assert.NotEmpty(t, res)
// 		assert.Empty(t, res[0].Result().Error)
// 		assert.True(t, res[0].Data.Value.(int64) == 2)
// 	})

// 	t.Run("ask details about an iam credential report entry", func(t *testing.T) {
// 		res := testQuery(t, "aws.iam.credentialreport[0]['user']")
// 		assert.NotEmpty(t, res)
// 		assert.Empty(t, res[0].Result().Error)
// 		assert.True(t, res[0].Data.Value.(string) == "<root_account>")
// 	})

// 	t.Run("use where for credential report", func(t *testing.T) {
// 		res := testQuery(t, "aws.iam.credentialreport.where( _['user'] == '<root_account>')")
// 		assert.NotEmpty(t, res)
// 		assert.Empty(t, res[0].Result().Error)
// 		assert.True(t, res[0].Data.Value.(string) == "<root_account>")
// 	})
// }

// func TestResource_AwsIamAccountSummary(t *testing.T) {
// 	t.Run("run a aws iam credential report", func(t *testing.T) {
// 		res := testQuery(t, "aws.iam.accountsummary")
// 		assert.NotEmpty(t, res)
// 		assert.Empty(t, res[0].Result().Error)
// 	})
// }

// func TestAccountSummary(t *testing.T) {

// 	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile("mondoo-inc"))
// 	if err != nil {
// 		panic(err)
// 	}
// 	cfg.Region = endpoints.UsEast1RegionID

// 	// iterate over each region?
// 	svc := iam.New(cfg)
// 	require.NoError(t, err)
// 	ctx := context.Background()
// 	// _, err = svc.GetAccountPasswordPolicyRequest(&iam.GetAccountPasswordPolicyInput{}).Send(ctx)
// 	// require.NoError(t, err)

// 	// summaryResp, err := svc.GetAccountSummaryRequest(&iam.GetAccountSummaryInput{}).Send(ctx)
// 	// require.NoError(t, err)
// 	// fmt.Printf("%v", summaryResp)

// 	// usersResp, err := svc.ListUsersRequest(&iam.ListUsersInput{}).Send(ctx)
// 	// require.NoError(t, err)
// 	// fmt.Printf("%v", usersResp)

// 	// userAttachedPolicies, err := svc.ListAttachedUserPoliciesRequest(&iam.ListAttachedUserPoliciesInput{}).Send(ctx)
// 	// require.NoError(t, err)
// 	// fmt.Printf("%v", userAttachedPolicies)

// 	// userPolicies, err := svc.ListUserPoliciesRequest(&iam.ListUserPoliciesInput{}).Send(ctx)
// 	// require.NoError(t, err)
// 	// fmt.Printf("%v", userPolicies)

// 	// userPolicies, err := svc.ListUserPoliciesRequest(&iam.ListUserPoliciesInput{}).Send(ctx)
// 	// require.NoError(t, err)
// 	// fmt.Printf("%v", userPolicies)

// 	// devicesResp, err := svc.ListVirtualMFADevicesRequest(&iam.ListVirtualMFADevicesInput{}).Send(ctx)
// 	// require.NoError(t, err)
// 	// fmt.Printf("%v", devicesResp)
// 	policyArn := "arn:aws:iam::aws:policy/AWSDirectConnectReadOnlyAccess"
// 	policyVersion := "v4"
// 	policydocResp, err := svc.GetPolicyVersionRequest(&iam.GetPolicyVersionInput{
// 		PolicyArn: &policyArn,
// 		VersionId: &policyVersion,
// 	}).Send(ctx)
// 	require.NoError(t, err)
// 	fmt.Printf("%v", policydocResp)

// 	decodedValue, err := url.QueryUnescape(*policydocResp.PolicyVersion.Document)
// 	require.NoError(t, err)
// 	assert.Equal(t, "", decodedValue)

// }
