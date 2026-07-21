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

func TestGetAssetIdentifierForService(t *testing.T) {
	const (
		s3Arn  = "arn:aws:s3:::my-bucket"
		mskArn = "arn:aws:kafka:us-east-1:012345678910:cluster/my-cluster/abc-123"
	)

	t.Run("returns the arn when the service matches", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("my-cluster", []string{mskArn})
		assert.Equal(t, mskArn, getAssetIdentifierForService(runtime, "kafka"))
	})

	t.Run("returns the arn when the service matches any of several", func(t *testing.T) {
		// Some resources legitimately accept more than one service token, e.g.
		// DocumentDB and Neptune assets both carry rds ARNs.
		runtime := testAwsIdentifierRuntime("my-cluster", []string{mskArn})
		assert.Equal(t, mskArn, getAssetIdentifierForService(runtime, "rds", "kafka", "dynamodb"))
	})

	t.Run("returns empty when the service does not match", func(t *testing.T) {
		// The regression this function exists for: a Route53 hosted zone asset
		// must not be adopted as, say, an aws.msk.cluster ARN just because the
		// init ran with no args.
		runtime := testAwsIdentifierRuntime("example.com.", []string{"arn:aws:route53:::hostedzone/Z123456"})
		assert.Empty(t, getAssetIdentifierForService(runtime, "kafka"))
	})

	t.Run("returns empty when none of several services match", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("my-bucket", []string{s3Arn})
		assert.Empty(t, getAssetIdentifierForService(runtime, "kafka", "rds", "dynamodb"))
	})

	t.Run("returns empty when no service is requested", func(t *testing.T) {
		// A caller that forgets its service token gets no ARN, not every ARN.
		runtime := testAwsIdentifierRuntime("my-bucket", []string{s3Arn})
		assert.Empty(t, getAssetIdentifierForService(runtime))
	})

	t.Run("matches the service exactly, not by prefix or substring", func(t *testing.T) {
		// "ec2" must not be satisfied by "ec2messages" or vice versa, and a
		// truncated token must not match a longer real one.
		runtime := testAwsIdentifierRuntime("i-abc", []string{"arn:aws:ec2:us-east-1:012345678910:instance/i-abc"})
		assert.Equal(t, "arn:aws:ec2:us-east-1:012345678910:instance/i-abc", getAssetIdentifierForService(runtime, "ec2"))
		assert.Empty(t, getAssetIdentifierForService(runtime, "ec"))
		assert.Empty(t, getAssetIdentifierForService(runtime, "ec2messages"))
	})

	t.Run("matches the service case-sensitively", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("my-bucket", []string{s3Arn})
		assert.Empty(t, getAssetIdentifierForService(runtime, "S3"))
	})

	t.Run("matches global arns that carry no region or account", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("my-bucket", []string{s3Arn})
		assert.Equal(t, s3Arn, getAssetIdentifierForService(runtime, "s3"))
	})

	t.Run("returns empty when the connection is not an AwsConnection", func(t *testing.T) {
		assert.Empty(t, getAssetIdentifierForService(&plugin.Runtime{}, "s3"))
	})

	t.Run("returns empty when the asset has no arn", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("some-asset", []string{"not-an-arn"})
		assert.Empty(t, getAssetIdentifierForService(runtime, "s3"))
	})

	t.Run("returns empty when the only arn fails to parse", func(t *testing.T) {
		// An unparseable ARN is dropped by getAssetIdentifier, so there is
		// nothing to service-match against.
		runtime := testAwsIdentifierRuntime("AWS Account 010438491650", []string{"arn:aws:sts::010438491650"})
		assert.Empty(t, getAssetIdentifierForService(runtime, "sts"))
	})
}

func TestResolveArnArg(t *testing.T) {
	const (
		mskArn = "arn:aws:kafka:us-east-1:012345678910:cluster/my-cluster/abc-123"
		s3Arn  = "arn:aws:s3:::my-bucket"
	)

	// --- adoption path: args empty, ARN comes from the scanned asset ---

	t.Run("adopts the asset arn when the service matches", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("my-cluster", []string{mskArn})
		args := map[string]*llx.RawData{}
		got, err := resolveArnArg(runtime, args, "aws msk cluster", "kafka")
		require.NoError(t, err)
		assert.Equal(t, mskArn, got)
		assert.Equal(t, mskArn, args["arn"].Value.(string))
	})

	t.Run("errors when the asset belongs to another service", func(t *testing.T) {
		// A Route53 hosted zone asset must not be adopted as an MSK cluster ARN
		// just because the init ran with no args.
		runtime := testAwsIdentifierRuntime("example.com.", []string{"arn:aws:route53:::hostedzone/Z123456"})
		args := map[string]*llx.RawData{}
		_, err := resolveArnArg(runtime, args, "aws msk cluster", "kafka")
		require.ErrorContains(t, err, "arn required to fetch aws msk cluster")
		assert.Nil(t, args["arn"])
	})

	t.Run("errors when there is no asset arn", func(t *testing.T) {
		args := map[string]*llx.RawData{}
		_, err := resolveArnArg(&plugin.Runtime{}, args, "aws msk cluster", "kafka")
		require.ErrorContains(t, err, "arn required to fetch aws msk cluster")
	})

	t.Run("does not adopt the asset arn when args are already populated", func(t *testing.T) {
		// Only the no-args form adopts; an explicit lookup must not be silently
		// retargeted at the scanned asset.
		runtime := testAwsIdentifierRuntime("my-cluster", []string{mskArn})
		args := map[string]*llx.RawData{"name": llx.StringData("other-cluster")}
		_, err := resolveArnArg(runtime, args, "aws msk cluster", "kafka")
		require.ErrorContains(t, err, "arn required to fetch aws msk cluster")
		assert.Nil(t, args["arn"])
	})

	// --- validation path: ARN supplied explicitly by the caller ---

	t.Run("accepts an explicit arn of the wanted service", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData(mskArn)}
		got, err := resolveArnArg(runtime, args, "aws msk cluster", "kafka")
		require.NoError(t, err)
		assert.Equal(t, mskArn, got)
		assert.Equal(t, mskArn, args["arn"].Value.(string))
	})

	t.Run("returns an empty arn alongside every error", func(t *testing.T) {
		// Callers use the returned ARN directly, so a failed call must not hand
		// back a usable-looking value.
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		for _, bad := range []*llx.RawData{
			llx.StringData(s3Arn),        // wrong service
			llx.StringData("not-an-arn"), // not an arn at all
			llx.IntData(42),              // not a string
		} {
			args := map[string]*llx.RawData{"arn": bad}
			got, err := resolveArnArg(runtime, args, "aws msk cluster", "kafka")
			require.Error(t, err)
			assert.Empty(t, got)
		}
	})

	t.Run("accepts an explicit arn matching any of several services", func(t *testing.T) {
		// e.g. documentdb and neptune assets both carry rds ARNs.
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData(mskArn)}
		_, err := resolveArnArg(runtime, args, "documentdb cluster", "rds", "kafka")
		require.NoError(t, err)
	})

	t.Run("rejects an explicit arn from another service", func(t *testing.T) {
		// The case the len(args)==0 gate structurally cannot see.
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData("arn:aws:lambda:us-east-1:012345678910:function:my-fn:PROD")}
		_, err := resolveArnArg(runtime, args, "cloudwatch log group", "logs")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not a cloudwatch log group arn")
		assert.Contains(t, err.Error(), `service is "lambda"`)
		assert.Contains(t, err.Error(), "expected logs")
	})

	t.Run("names every wanted service in the mismatch error", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData(s3Arn)}
		_, err := resolveArnArg(runtime, args, "documentdb cluster", "rds", "kafka")
		require.ErrorContains(t, err, "expected rds or kafka")
	})

	t.Run("rejects a malformed arn", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData("arn:aws:sts::010438491650")}
		_, err := resolveArnArg(runtime, args, "ec2 instance", "sts")
		require.ErrorContains(t, err, "invalid arn")
	})

	t.Run("rejects a reference that is not an arn", func(t *testing.T) {
		// The hardened contract: resources that accept a non-ARN reference (a
		// bare KMS key ID, an "alias/..." name, a bare resource name) must not
		// use resolveArnArg at all -- here every one of them is an error.
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		for _, ref := range []string{"1234abcd-12ab-34cd-56ef-1234567890ab", "alias/my-key", "my-bucket", ""} {
			args := map[string]*llx.RawData{"arn": llx.StringData(ref)}
			_, err := resolveArnArg(runtime, args, "kms key", "kms")
			require.Error(t, err, "ref %q", ref)
		}
	})

	t.Run("rejects every arn when no service is wanted", func(t *testing.T) {
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.StringData(s3Arn)}
		_, err := resolveArnArg(runtime, args, "s3 bucket")
		require.Error(t, err)
	})

	t.Run("errors instead of panicking on a non-string arn", func(t *testing.T) {
		// Most inits go on to assert args["arn"].Value.(string) bare; a panic
		// there would take down the whole scan.
		runtime := testAwsIdentifierRuntime("irrelevant", nil)
		args := map[string]*llx.RawData{"arn": llx.IntData(42)}
		_, err := resolveArnArg(runtime, args, "aws msk cluster", "kafka")
		require.ErrorContains(t, err, "arn for aws msk cluster must be a string")
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
