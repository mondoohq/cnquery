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
// Both paths it exercises fail inside arnSpec.resolve, before the init builds an
// AWS client or issues a request, so the whole table runs offline with no
// credentials.
//
// What it protects: an init that drops the arnSpec.resolve call (or reverts to
// adopting the asset ARN unchecked) starts resolving a foreign ARN into a husk
// resource whose fields are never set -- which surfaces to users as "provider
// returned no data and no error for a field" and "llx: encountered a primitive
// with no type information", with no attribution back to the init at fault.
func TestInitsRejectForeignArns(t *testing.T) {
	// No resource in the table accepts iam, so this stands in for "an ARN from
	// some other service" for all of them.
	const foreignArn = "arn:aws:iam::012345678910:role/some-role"

	// Pairing each init with its spec keeps the expected resource name in the
	// error messages tied to the single place that declares it.
	cases := []struct {
		init arnInitFunc
		spec arnSpec
	}{
		{initAwsApigatewayRestapi, apigatewayRestapiArnSpec},
		{initAwsBatchJobDefinition, batchJobDefinitionArnSpec},
		{initAwsCloudfrontDistribution, cloudfrontDistributionArnSpec},
		{initAwsCloudwatchLoggroup, cloudwatchLoggroupArnSpec},
		{initAwsCognitoUserPool, cognitoUserPoolArnSpec},
		{initAwsDocumentdbCluster, documentdbClusterArnSpec},
		{initAwsDocumentdbInstance, documentdbInstanceArnSpec},
		{initAwsDynamodbGlobaltable, dynamodbGlobaltableArnSpec},
		{initAwsDynamodbTable, dynamodbTableArnSpec},
		{initAwsEc2Instance, ec2InstanceArnSpec},
		{initAwsEc2Volume, ec2VolumeArnSpec},
		{initAwsEcrImage, ecrImageArnSpec},
		{initAwsEcsCluster, ecsClusterArnSpec},
		{initAwsEcsService, ecsServiceArnSpec},
		{initAwsEcsTask, ecsTaskArnSpec},
		{initAwsEcsTaskDefinition, ecsTaskDefinitionArnSpec},
		{initAwsEfsFilesystem, efsFilesystemArnSpec},
		{initAwsEksCluster, eksClusterArnSpec},
		{initAwsElasticacheCluster, elasticacheClusterArnSpec},
		{initAwsElbLoadbalancer, elbLoadbalancerArnSpec},
		{initAwsEmrCluster, emrClusterArnSpec},
		{initAwsFsxBackup, fsxBackupArnSpec},
		{initAwsFsxCache, fsxCacheArnSpec},
		{initAwsFsxFilesystem, fsxFilesystemArnSpec},
		{initAwsMemorydbCluster, memorydbClusterArnSpec},
		{initAwsMskCluster, mskClusterArnSpec},
		{initAwsNeptuneCluster, neptuneClusterArnSpec},
		{initAwsRdsDbcluster, rdsDbclusterArnSpec},
		{initAwsRdsDbinstance, rdsDbinstanceArnSpec},
		{initAwsRedshiftCluster, redshiftClusterArnSpec},
		{initAwsSagemakerDomain, sagemakerDomainArnSpec},
		{initAwsSagemakerModel, sagemakerModelArnSpec},
		{initAwsSagemakerNotebookinstance, sagemakerNotebookinstanceArnSpec},
		{initAwsSecretsmanagerSecret, secretsmanagerSecretArnSpec},
		{initAwsSsmInstance, ssmInstanceArnSpec},
		{initAwsStoragegatewayGateway, storagegatewayGatewayArnSpec},
		{initAwsTransferServer, transferServerArnSpec},
	}

	for _, tc := range cases {
		t.Run(tc.spec.resource, func(t *testing.T) {
			t.Run("rejects an explicitly passed foreign arn", func(t *testing.T) {
				runtime := testAwsIdentifierRuntime("irrelevant", nil)
				args := map[string]*llx.RawData{"arn": llx.StringData(foreignArn)}

				_, res, err := tc.init(runtime, args)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "is not an "+tc.spec.resource+" arn")
				assert.Nil(t, res, "must not build a resource from a foreign arn")
			})

			t.Run("does not adopt an asset arn from another service", func(t *testing.T) {
				// A bare singular query (`aws.msk.cluster` with no args) while an
				// unrelated asset is being scanned must not adopt that asset's ARN.
				runtime := testAwsIdentifierRuntime("irrelevant", []string{foreignArn})
				args := map[string]*llx.RawData{}

				_, res, err := tc.init(runtime, args)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "arn required to fetch "+tc.spec.resource)
				assert.Nil(t, res, "must not build a resource from an unrelated asset")
				assert.Nil(t, args["arn"], "must not leave a foreign arn in args")
			})
		})
	}
}
