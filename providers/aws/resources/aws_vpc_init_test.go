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
	"go.mondoo.com/mql/v13/utils/syncx"
)

// testAwsRuntime creates a runtime with an AwsConnection whose asset has
// the given name and platform IDs.  It also pre-populates the resource
// cache with an "aws" resource whose Vpcs list contains the supplied VPCs,
// so initAwsVpc can resolve them without making real API calls.
func testAwsRuntime(assetName string, platformIds []string, vpcs []*mqlAwsVpc) *plugin.Runtime {
	asset := &inventory.Asset{
		Name:        assetName,
		PlatformIds: platformIds,
		Connections: []*inventory.Config{{}},
	}

	conn := &connection.AwsConnection{
		Connection: plugin.NewConnection(1, asset),
		Conf: &inventory.Config{
			Discover: &inventory.Discovery{},
		},
	}
	conn.UpdateAsset(asset)

	resources := &syncx.Map[plugin.Resource]{}
	runtime := &plugin.Runtime{
		Connection: conn,
		Resources:  resources,
	}

	// Pre-populate the "aws" resource with VPCs already loaded.
	awsRes := &mqlAws{MqlRuntime: runtime}
	vpcList := make([]any, len(vpcs))
	for i, v := range vpcs {
		vpcList[i] = v
	}
	awsRes.Vpcs = plugin.TValue[[]any]{Data: vpcList, State: plugin.StateIsSet}
	// Cache key: "aws" + \x00 + __id (empty)
	resources.Set("aws\x00", awsRes)

	return runtime
}

func TestInitAwsVpc(t *testing.T) {
	const (
		vpcId  = "vpc-0abc123def456"
		vpcArn = "arn:aws:vpc:us-east-1:123456789012:id/vpc-0abc123def456"
		region = "us-east-1"
	)

	testVpc := &mqlAwsVpc{
		Id:     plugin.TValue[string]{Data: vpcId, State: plugin.StateIsSet},
		Arn:    plugin.TValue[string]{Data: vpcArn, State: plugin.StateIsSet},
		Region: plugin.TValue[string]{Data: region, State: plugin.StateIsSet},
	}

	t.Run("derives VPC id from ARN when asset name is a tag (not VPC ID)", func(t *testing.T) {
		// This is the bug scenario: during discovery, MqlObjectToAsset
		// overrides the asset name with the VPC's "Name" tag.  The old
		// code set args["id"] = ids.name (the tag), then tried to match
		// against vpc.Id.Data (the real VPC ID) — which never matched.
		// deriveVpcTarget must derive the id from the ARN, never the name.
		tagName := "scottford-upgrade-eks-container-escape-demo-02c1-vpc"
		runtime := testAwsRuntime(
			tagName,
			[]string{
				"//platformid.api.mondoo.app/runtime/aws/accounts/123456789012/regions/us-east-1/vpcs/" + vpcId,
				vpcArn,
			},
			[]*mqlAwsVpc{testVpc},
		)

		args := map[string]*llx.RawData{}
		region, gotVpcId := deriveVpcTarget(runtime, args)
		assert.Equal(t, "us-east-1", region)
		assert.Equal(t, vpcId, gotVpcId)
		assert.Equal(t, vpcArn, args["arn"].Value.(string))
	})

	t.Run("derives VPC id from ARN when asset name matches VPC ID", func(t *testing.T) {
		// When there's no "Name" tag, the asset name IS the VPC ID.
		runtime := testAwsRuntime(
			vpcId,
			[]string{
				"//platformid.api.mondoo.app/runtime/aws/accounts/123456789012/regions/us-east-1/vpcs/" + vpcId,
				vpcArn,
			},
			[]*mqlAwsVpc{testVpc},
		)

		args := map[string]*llx.RawData{}
		region, gotVpcId := deriveVpcTarget(runtime, args)
		assert.Equal(t, "us-east-1", region)
		assert.Equal(t, vpcId, gotVpcId)
	})

	t.Run("derives region and id from an explicit arn arg", func(t *testing.T) {
		runtime := testAwsRuntime("irrelevant", nil, []*mqlAwsVpc{testVpc})

		args := map[string]*llx.RawData{"arn": llx.StringData(vpcArn)}
		region, gotVpcId := deriveVpcTarget(runtime, args)
		assert.Equal(t, "us-east-1", region)
		assert.Equal(t, vpcId, gotVpcId)
	})

	t.Run("resolves VPC by explicit id arg", func(t *testing.T) {
		runtime := testAwsRuntime("irrelevant", nil, []*mqlAwsVpc{testVpc})

		args := map[string]*llx.RawData{
			"id": llx.StringData(vpcId),
		}
		_, resource, err := initAwsVpc(runtime, args)
		require.NoError(t, err)
		require.NotNil(t, resource)

		vpc := resource.(*mqlAwsVpc)
		assert.Equal(t, vpcId, vpc.Id.Data)
	})

	t.Run("returns error when VPC not found", func(t *testing.T) {
		runtime := testAwsRuntime("irrelevant", nil, []*mqlAwsVpc{testVpc})

		args := map[string]*llx.RawData{
			"id": llx.StringData("vpc-doesnotexist"),
		}
		_, _, err := initAwsVpc(runtime, args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "vpc does not exist")
	})

	t.Run("returns error when asset has no valid ARN in platform IDs", func(t *testing.T) {
		// getAssetIdentifier returns "" when the asset has no valid
		// "arn:aws:" platform ID, so no arn arg is injected and the init
		// errors out before any lookup.
		runtime := testAwsRuntime("some-vpc", []string{"not-an-arn"}, []*mqlAwsVpc{testVpc})

		args := map[string]*llx.RawData{}
		_, _, err := initAwsVpc(runtime, args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "arn or id required")
	})
}
