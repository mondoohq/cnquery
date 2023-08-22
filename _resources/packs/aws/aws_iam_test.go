// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package aws_test

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/resources/packs/aws"
)

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
// 	// _, err = svc.GetAccountPasswordPolicy(ctx, &iam.GetAccountPasswordPolicyInput{})
// 	// require.NoError(t, err)

// 	// summaryResp, err := svc.GetAccountSummary(ctx, &iam.GetAccountSummaryInput{})
// 	// require.NoError(t, err)
// 	// fmt.Printf("%v", summaryResp)

// 	// usersResp, err := svc.ListUsers(ctx, &iam.ListUsersInput{})
// 	// require.NoError(t, err)
// 	// fmt.Printf("%v", usersResp)

// 	// userAttachedPolicies, err := svc.ListAttachedUserPolicies(ctx, &iam.ListAttachedUserPoliciesInput{})
// 	// require.NoError(t, err)
// 	// fmt.Printf("%v", userAttachedPolicies)

// 	// userPolicies, err := svc.ListUserPolicies(ctx, &iam.ListUserPoliciesInput{})
// 	// require.NoError(t, err)
// 	// fmt.Printf("%v", userPolicies)

// 	// userPolicies, err := svc.ListUserPolicies(ctx, &iam.ListUserPoliciesInput{})
// 	// require.NoError(t, err)
// 	// fmt.Printf("%v", userPolicies)

// 	// devicesResp, err := svc.ListVirtualMFADevices(ctx, &iam.ListVirtualMFADevicesInput{})
// 	// require.NoError(t, err)
// 	// fmt.Printf("%v", devicesResp)
// 	policyArn := "arn:aws:iam::aws:policy/AWSDirectConnectReadOnlyAccess"
// 	policyVersion := "v4"
// 	policydocResp, err := svc.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
// 		PolicyArn: &policyArn,
// 		VersionId: &policyVersion,
// 	})
// 	require.NoError(t, err)
// 	fmt.Printf("%v", policydocResp)

// 	decodedValue, err := url.QueryUnescape(*policydocResp.PolicyVersion.Document)
// 	require.NoError(t, err)
// 	assert.Equal(t, "", decodedValue)

// }

func TestParseAwsIso8601Parser(t *testing.T) {
	timestamps := []string{
		"2019-06-11T19:04:54+00:00",
		"2019-08-08T10:36:33+00:00",
	}

	for i := range timestamps {
		format := "2006-01-02T15:04:05-07:00"
		_, err := time.Parse(format, timestamps[i])
		require.NoError(t, err)
	}
}

func TestParsePasswordPolicy(t *testing.T) {
	pPolicy := &types.PasswordPolicy{}
	assert.Equal(t, map[string]interface{}{"AllowUsersToChangePassword": false, "ExpirePasswords": false, "HardExpiry": false, "MaxPasswordAge": "0", "MinimumPasswordLength": "0", "PasswordReusePrevention": "0", "RequireLowercaseCharacters": false, "RequireNumbers": false, "RequireSymbols": false, "RequireUppercaseCharacters": false}, aws.ParsePasswordPolicy(pPolicy))
	pPolicy.AllowUsersToChangePassword = true
	pPolicy.RequireNumbers = true
	assert.Equal(t, map[string]interface{}{"AllowUsersToChangePassword": true, "ExpirePasswords": false, "HardExpiry": false, "MaxPasswordAge": "0", "MinimumPasswordLength": "0", "PasswordReusePrevention": "0", "RequireLowercaseCharacters": false, "RequireNumbers": true, "RequireSymbols": false, "RequireUppercaseCharacters": false}, aws.ParsePasswordPolicy(pPolicy))
}
