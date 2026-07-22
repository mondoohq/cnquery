// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func testAwsIdentifierRuntime(assetName string, platformIds []string) *plugin.Runtime {
	asset := &inventory.Asset{
		Name:        assetName,
		PlatformIds: platformIds,
		Connections: []*inventory.Config{{}},
	}
	conn := &connection.AwsConnection{
		Connection: plugin.NewConnection(1, asset),
	}
	conn.UpdateAsset(asset)
	return &plugin.Runtime{Connection: conn}
}

func TestGetAssetIdentifier(t *testing.T) {
	const validArn = "arn:aws:s3:::my-bucket"

	t.Run("returns empty when the connection is not an AwsConnection", func(t *testing.T) {
		assert.Empty(t, getAssetIdentifier(&plugin.Runtime{}))
	})

	t.Run("returns empty when the connection has no asset", func(t *testing.T) {
		conn := &connection.AwsConnection{
			Connection: plugin.NewConnection(1, &inventory.Asset{Connections: []*inventory.Config{{}}}),
		}
		// no UpdateAsset call, so conn.Asset() returns nil
		assert.Empty(t, getAssetIdentifier(&plugin.Runtime{Connection: conn}))
	})

	t.Run("returns the arn for a valid arn:aws: platform ID", func(t *testing.T) {
		assert.Equal(t, validArn, getAssetIdentifier(testAwsIdentifierRuntime("my-bucket", []string{validArn})))
	})

	t.Run("returns empty when the only arn:aws: platform ID fails to parse", func(t *testing.T) {
		// Regression: an AWS account asset can carry a malformed STS ARN
		// ("arn: not enough sections"). Returning it (or an empty string that
		// callers inject anyway) made inits set args["arn"] = "" and end up
		// creating blank resources with unset fields.
		assert.Empty(t, getAssetIdentifier(testAwsIdentifierRuntime("AWS Account 010438491650", []string{"arn:aws:sts::010438491650"})))
	})

	t.Run("returns empty when no platform ID has the arn:aws: prefix", func(t *testing.T) {
		assert.Empty(t, getAssetIdentifier(testAwsIdentifierRuntime("some-asset", []string{
			"//platformid.api.mondoo.app/runtime/aws/accounts/010438491650",
			"not-an-arn",
		})))
	})

	t.Run("returns empty when the asset has no platform IDs", func(t *testing.T) {
		assert.Empty(t, getAssetIdentifier(testAwsIdentifierRuntime("some-asset", nil)))
	})

	t.Run("skips an invalid ARN and keeps the valid one", func(t *testing.T) {
		assert.Equal(t, validArn, getAssetIdentifier(testAwsIdentifierRuntime("my-bucket", []string{
			"arn:aws:sts::010438491650",
			validArn,
		})))
	})

	t.Run("last valid ARN wins when multiple are present", func(t *testing.T) {
		first := "arn:aws:s3:::first-bucket"
		second := "arn:aws:s3:::second-bucket"
		assert.Equal(t, second, getAssetIdentifier(testAwsIdentifierRuntime("my-bucket", []string{first, second})))
	})
}

const (
	testMskArn = "arn:aws:kafka:us-east-1:012345678910:cluster/my-cluster/abc-123"
	testS3Arn  = "arn:aws:s3:::my-bucket"
)

var testMskSpec = arnSpec{resource: ResourceAwsMskCluster, services: []string{"kafka"}}

func TestArnSpecAssetRef(t *testing.T) {
	t.Run("returns the arn when the service matches", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("my-cluster", []string{testMskArn})
		assert.Equal(t, testMskArn, testMskSpec.assetRef(runtime).RawArn)
	})

	t.Run("returns the arn when the service matches any of several", func(t *testing.T) {
		// Some resources legitimately accept more than one service token, e.g.
		// ecr repositories carry either an ecr or an ecr-public ARN.
		spec := arnSpec{resource: ResourceAwsMskCluster, services: []string{"rds", "kafka", "dynamodb"}}
		runtime := testAwsIdentifierRuntime("my-cluster", []string{testMskArn})
		assert.Equal(t, testMskArn, spec.assetRef(runtime).RawArn)
	})

	t.Run("returns empty when the service does not match", func(t *testing.T) {
		// The regression the service gate exists for: a Route53 hosted zone
		// asset must not be adopted as, say, an aws.msk.cluster ARN just because
		// the init ran with no args.
		runtime := testAwsIdentifierRuntime("example.com.", []string{"arn:aws:route53:::hostedzone/Z123456"})
		assert.Empty(t, testMskSpec.assetRef(runtime).RawArn)
	})

	t.Run("returns empty when none of several services match", func(t *testing.T) {
		spec := arnSpec{resource: ResourceAwsMskCluster, services: []string{"kafka", "rds", "dynamodb"}}
		runtime := testAwsIdentifierRuntime("my-bucket", []string{testS3Arn})
		assert.Empty(t, spec.assetRef(runtime).RawArn)
	})

	t.Run("returns empty when the spec wants no service", func(t *testing.T) {
		// A spec that forgets its service token gets no ARN, not every ARN.
		spec := arnSpec{resource: ResourceAwsS3Bucket}
		runtime := testAwsIdentifierRuntime("my-bucket", []string{testS3Arn})
		assert.Empty(t, spec.assetRef(runtime).RawArn)
	})

	t.Run("matches the service exactly, not by prefix or substring", func(t *testing.T) {
		// "ec2" must not be satisfied by "ec2messages" or vice versa, and a
		// truncated token must not match a longer real one.
		const ec2Arn = "arn:aws:ec2:us-east-1:012345678910:instance/i-abc"
		runtime := testAwsIdentifierRuntime("i-abc", []string{ec2Arn})
		assert.Equal(t, ec2Arn, arnSpec{resource: ResourceAwsEc2Instance, services: []string{"ec2"}}.assetRef(runtime).RawArn)
		assert.Empty(t, arnSpec{resource: ResourceAwsEc2Instance, services: []string{"ec"}}.assetRef(runtime).RawArn)
		assert.Empty(t, arnSpec{resource: ResourceAwsEc2Instance, services: []string{"ec2messages"}}.assetRef(runtime).RawArn)
	})

	t.Run("matches the service case-sensitively", func(t *testing.T) {
		spec := arnSpec{resource: ResourceAwsS3Bucket, services: []string{"S3"}}
		runtime := testAwsIdentifierRuntime("my-bucket", []string{testS3Arn})
		assert.Empty(t, spec.assetRef(runtime).RawArn)
	})

	t.Run("matches global arns that carry no region or account", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("my-bucket", []string{testS3Arn})
		assert.Equal(t, testS3Arn, s3BucketArnSpec.assetRef(runtime).RawArn)
	})

	t.Run("returns empty when the connection is not an AwsConnection", func(t *testing.T) {
		assert.Empty(t, s3BucketArnSpec.assetRef(&plugin.Runtime{}).RawArn)
	})

	t.Run("returns empty when the asset has no arn", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("some-asset", []string{"not-an-arn"})
		assert.Empty(t, s3BucketArnSpec.assetRef(runtime).RawArn)
	})

	t.Run("returns empty when the only arn fails to parse", func(t *testing.T) {
		// An unparseable ARN is dropped by getAssetIdentifier, so there is
		// nothing to service-match against.
		spec := arnSpec{resource: ResourceAwsEc2Instance, services: []string{"sts"}}
		runtime := testAwsIdentifierRuntime("AWS Account 010438491650", []string{"arn:aws:sts::010438491650"})
		assert.Empty(t, spec.assetRef(runtime).RawArn)
	})

	t.Run("rejects an asset arn whose resource segment fails the prefix", func(t *testing.T) {
		// aws.codebuild.project only ever wants a project ARN; a codebuild
		// report-group asset must not be adopted in its place.
		runtime := testAwsIdentifierRuntime("my-reports", []string{"arn:aws:codebuild:us-east-1:012345678910:report-group/my-reports"})
		assert.Empty(t, codebuildProjectArnSpec.assetRef(runtime).RawArn)
	})

	t.Run("trims the prefix off ResourceID", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("my-project", []string{"arn:aws:codebuild:us-east-1:012345678910:project/my-project"})
		ref := codebuildProjectArnSpec.assetRef(runtime)
		assert.Equal(t, "my-project", ref.ResourceID)
		assert.Equal(t, "us-east-1", ref.Region)
	})
}

func TestArnSpecResolve(t *testing.T) {
	// --- adoption path: args empty, ARN comes from the scanned asset ---

	t.Run("adopts the asset arn when the service matches", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("my-cluster", []string{testMskArn})
		args := map[string]*llx.RawData{}
		got, err := testMskSpec.resolve(runtime, args)
		require.NoError(t, err)
		assert.Equal(t, testMskArn, got.RawArn)
		assert.Equal(t, testMskArn, args["arn"].Value.(string))
	})

	t.Run("errors when the asset belongs to another service", func(t *testing.T) {
		// A Route53 hosted zone asset must not be adopted as an MSK cluster ARN
		// just because the init ran with no args.
		runtime := testAwsIdentifierRuntime("example.com.", []string{"arn:aws:route53:::hostedzone/Z123456"})
		args := map[string]*llx.RawData{}
		_, err := testMskSpec.resolve(runtime, args)
		require.ErrorContains(t, err, "arn required to fetch aws.msk.cluster")
		assert.Nil(t, args["arn"])
	})

	t.Run("errors when there is no asset arn", func(t *testing.T) {
		args := map[string]*llx.RawData{}
		_, err := testMskSpec.resolve(&plugin.Runtime{}, args)
		require.ErrorContains(t, err, "arn required to fetch aws.msk.cluster")
	})

	t.Run("does not adopt the asset arn when args are already populated", func(t *testing.T) {
		// Only the no-args form adopts; an explicit lookup must not be silently
		// retargeted at the scanned asset.
		runtime := testAwsIdentifierRuntime("my-cluster", []string{testMskArn})
		args := map[string]*llx.RawData{"name": llx.StringData("other-cluster")}
		_, err := testMskSpec.resolve(runtime, args)
		require.ErrorContains(t, err, "arn required to fetch aws.msk.cluster")
		assert.Nil(t, args["arn"])
	})

	// --- validation path: ARN supplied explicitly by the caller ---

	t.Run("accepts an explicit arn of the wanted service", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData(testMskArn)}
		got, err := testMskSpec.resolve(runtime, args)
		require.NoError(t, err)
		assert.Equal(t, testMskArn, got.RawArn)
		assert.Equal(t, "us-east-1", got.Region)
		assert.Equal(t, testMskArn, args["arn"].Value.(string))
	})

	t.Run("returns an empty ref alongside every error", func(t *testing.T) {
		// Callers use the returned ARN directly, so a failed call must not hand
		// back a usable-looking value.
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		for _, bad := range []*llx.RawData{
			llx.StringData(testS3Arn),    // wrong service
			llx.StringData("not-an-arn"), // not an arn at all
			llx.IntData(42),              // not a string
		} {
			args := map[string]*llx.RawData{"arn": bad}
			got, err := testMskSpec.resolve(runtime, args)
			require.Error(t, err)
			assert.Empty(t, got.RawArn)
			assert.Empty(t, got.Region)
		}
	})

	t.Run("accepts an explicit arn matching any of several services", func(t *testing.T) {
		// e.g. an ecr repository ARN may be ecr or ecr-public.
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData("arn:aws:ecr-public::012345678910:repository/my-repo")}
		_, err := ecrRepositoryArnSpec.resolve(runtime, args)
		require.NoError(t, err)
	})

	t.Run("rejects an explicit arn from another service", func(t *testing.T) {
		// The case the len(args)==0 gate structurally cannot see.
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData("arn:aws:lambda:us-east-1:012345678910:function:my-fn:PROD")}
		_, err := cloudwatchLoggroupArnSpec.resolve(runtime, args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not an aws.cloudwatch.loggroup arn")
		assert.Contains(t, err.Error(), `service is "lambda"`)
		assert.Contains(t, err.Error(), "expected logs")
	})

	t.Run("names every wanted service in the mismatch error", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData(testS3Arn)}
		_, err := ecrRepositoryArnSpec.resolve(runtime, args)
		require.ErrorContains(t, err, "expected ecr or ecr-public")
	})

	t.Run("rejects a malformed arn", func(t *testing.T) {
		spec := arnSpec{resource: ResourceAwsEc2Instance, services: []string{"sts"}}
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData("arn:aws:sts::010438491650")}
		_, err := spec.resolve(runtime, args)
		require.ErrorContains(t, err, "invalid arn")
	})

	t.Run("rejects a reference that is not an arn unless allowRef is set", func(t *testing.T) {
		// Bare KMS key IDs, "alias/..." names and bare resource names are only
		// legal in the arn arg of a spec that opts into them.
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		strict := arnSpec{resource: ResourceAwsKmsKey, services: []string{"kms"}}
		for _, ref := range []string{"1234abcd-12ab-34cd-56ef-1234567890ab", "alias/my-key", "my-bucket", ""} {
			args := map[string]*llx.RawData{"arn": llx.StringData(ref)}
			_, err := strict.resolve(runtime, args)
			require.Error(t, err, "ref %q", ref)

			args = map[string]*llx.RawData{"arn": llx.StringData(ref)}
			got, err := kmsKeyArnSpec.resolve(runtime, args)
			require.NoError(t, err, "ref %q", ref)
			assert.Equal(t, ref, got.RawArn)
		}
	})

	t.Run("still validates an arn-shaped value when allowRef is set", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData(testS3Arn)}
		_, err := kmsKeyArnSpec.resolve(runtime, args)
		require.ErrorContains(t, err, "is not an aws.kms.key arn")
	})

	t.Run("rejects every arn when the spec wants no service", func(t *testing.T) {
		spec := arnSpec{resource: ResourceAwsS3Bucket}
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData(testS3Arn)}
		_, err := spec.resolve(runtime, args)
		require.Error(t, err)
	})

	t.Run("errors instead of panicking on a non-string arn", func(t *testing.T) {
		// Unreachable in practice: the schema types arn as a string, and every
		// caller building one from a *string nil-checks first. This pins the
		// total behaviour so the guarantee survives a caller that does not.
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.IntData(42)}
		_, err := testMskSpec.resolve(runtime, args)
		require.ErrorContains(t, err, "arn for aws.msk.cluster must be a string")
	})

	// --- altKeys: resources that accept an identifier other than an ARN ---

	t.Run("succeeds with an empty ref when an altKey is supplied", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"name": llx.StringData("my-broker")}
		got, err := mqBrokerArnSpec.resolve(runtime, args)
		require.NoError(t, err)
		assert.Empty(t, got.RawArn)
	})

	t.Run("errors when neither the arn nor any altKey is supplied", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"region": llx.StringData("us-east-1")}
		_, err := mqBrokerArnSpec.resolve(runtime, args)
		require.ErrorContains(t, err, "arn or name required to fetch aws.mq.broker")
	})

	t.Run("lists every accepted key in the missing-key error", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"region": llx.StringData("us-east-1")}
		_, err := route53HostedZoneArnSpec.resolve(runtime, args)
		require.ErrorContains(t, err, "arn, id, or name required to fetch aws.route53.hostedZone")
	})

	t.Run("still validates the arn when an altKey is also supplied", func(t *testing.T) {
		// A wrong-service ARN is an error even though "name" alone would do.
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{
			"arn":  llx.StringData(testS3Arn),
			"name": llx.StringData("my-broker"),
		}
		_, err := mqBrokerArnSpec.resolve(runtime, args)
		require.ErrorContains(t, err, "is not an aws.mq.broker arn")
	})
}

func TestGetAssetName(t *testing.T) {
	t.Run("returns empty when the connection is not an AwsConnection", func(t *testing.T) {
		assert.Empty(t, getAssetName(&plugin.Runtime{}))
	})

	t.Run("returns empty when the connection has no asset", func(t *testing.T) {
		conn := &connection.AwsConnection{
			Connection: plugin.NewConnection(1, &inventory.Asset{Connections: []*inventory.Config{{}}}),
		}
		// no UpdateAsset call, so conn.Asset() returns nil
		assert.Empty(t, getAssetName(&plugin.Runtime{Connection: conn}))
	})

	t.Run("returns the asset name", func(t *testing.T) {
		assert.Equal(t, "my-user", getAssetName(testAwsIdentifierRuntime("my-user", nil)))
	})
}
