// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
