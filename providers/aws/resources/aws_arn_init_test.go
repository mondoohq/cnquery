// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type arnInitFunc func(*plugin.Runtime, map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)

// TestInitsRejectForeignArns covers every init that resolves strictly by ARN.
// Both paths it exercises fail inside resolveArnArg, before the init builds an
// AWS client or issues a request, so the whole table runs offline with no
// credentials.
//
// What it protects: an init that drops the resolveArnArg call (or reverts to
// adopting the asset ARN unchecked) starts resolving a foreign ARN into a husk
// resource whose fields are never set -- which surfaces to users as "provider
// returned no data and no error for a field" and "llx: encountered a primitive
// with no type information", with no attribution back to the init at fault.
func TestInitsRejectForeignArns(t *testing.T) {
	// No resource in the table accepts iam, so this stands in for "an ARN from
	// some other service" for all of them.
	const foreignArn = "arn:aws:iam::012345678910:role/some-role"

	cases := []struct {
		init arnInitFunc
		// resource is the name resolveArnArg puts in its error messages.
		resource string
	}{
		{initAwsApigatewayRestapi, "gateway restapi"},
		{initAwsBatchJobDefinition, "batch job definition"},
		{initAwsCloudfrontDistribution, "cloudfront distribution"},
		{initAwsCloudwatchLoggroup, "cloudwatch log group"},
		{initAwsCognitoUserPool, "cognito user pool"},
		{initAwsDocumentdbCluster, "documentdb cluster"},
		{initAwsDocumentdbInstance, "documentdb instance"},
		{initAwsDynamodbGlobaltable, "dynamodb global table"},
		{initAwsDynamodbTable, "dynamodb table"},
		{initAwsEc2Instance, "ec2 instance"},
		{initAwsEc2Volume, "aws volume"},
		{initAwsEcrImage, "ecr image"},
		{initAwsEcsCluster, "ecs cluster"},
		{initAwsEcsService, "ecs service"},
		{initAwsEcsTask, "ecs task"},
		{initAwsEcsTaskDefinition, "aws ecs task definition"},
		{initAwsEfsFilesystem, "efs filesystem"},
		{initAwsEksCluster, "eks cluster"},
		{initAwsElasticacheCluster, "elasticache cluster"},
		{initAwsElbLoadbalancer, "elb loadbalancer"},
		{initAwsEmrCluster, "emr cluster"},
		{initAwsFsxBackup, "fsx backup"},
		{initAwsFsxCache, "fsx cache"},
		{initAwsFsxFilesystem, "fsx filesystem"},
		{initAwsMemorydbCluster, "memorydb cluster"},
		{initAwsMskCluster, "aws msk cluster"},
		{initAwsNeptuneCluster, "neptune cluster"},
		{initAwsRdsDbcluster, "rds db cluster"},
		{initAwsRdsDbinstance, "rds db instance"},
		{initAwsRedshiftCluster, "redshift cluster"},
		{initAwsSagemakerDomain, "sagemaker domain"},
		{initAwsSagemakerModel, "sagemaker model"},
		{initAwsSagemakerNotebookinstance, "sagemaker notebookinstance"},
		{initAwsSecretsmanagerSecret, "secretsmanager secret"},
		{initAwsSsmInstance, "ssm instance"},
		{initAwsStoragegatewayGateway, "storage gateway"},
		{initAwsTransferServer, "transfer server"},
	}

	for _, tc := range cases {
		t.Run(tc.resource, func(t *testing.T) {
			t.Run("rejects an explicitly passed foreign arn", func(t *testing.T) {
				runtime := testAwsIdentifierRuntime("irrelevant", nil)
				args := map[string]*llx.RawData{"arn": llx.StringData(foreignArn)}

				_, res, err := tc.init(runtime, args)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "is not a "+tc.resource+" arn")
				assert.Nil(t, res, "must not build a resource from a foreign arn")
			})

			t.Run("does not adopt an asset arn from another service", func(t *testing.T) {
				// A bare singular query (`aws.msk.cluster` with no args) while an
				// unrelated asset is being scanned must not adopt that asset's ARN.
				runtime := testAwsIdentifierRuntime("irrelevant", []string{foreignArn})
				args := map[string]*llx.RawData{}

				_, res, err := tc.init(runtime, args)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "arn required to fetch "+tc.resource)
				assert.Nil(t, res, "must not build a resource from an unrelated asset")
				assert.Nil(t, args["arn"], "must not leave a foreign arn in args")
			})
		})
	}
}
