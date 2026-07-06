// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// TestElbWebAclNullForNonApplicationLB verifies that webAcl() short-circuits to a
// set-null value for every load balancer type that cannot carry a WAFv2 web ACL
// (Network, Gateway, Classic, or an unset type), without reaching the WAFv2 API.
func TestElbWebAclNullForNonApplicationLB(t *testing.T) {
	for _, elbType := range []string{"network", "gateway", "classic", ""} {
		t.Run("elbType="+elbType, func(t *testing.T) {
			lb := &mqlAwsElbLoadbalancer{}
			lb.ElbType = plugin.TValue[string]{Data: elbType, State: plugin.StateIsSet}
			result, err := lb.webAcl()
			require.NoError(t, err)
			require.Nil(t, result)
			assert.True(t, lb.WebAcl.IsNull())
			assert.True(t, lb.WebAcl.IsSet())
		})
	}
}

// TestWafAssociatedLoadBalancersEmptyForCloudFront verifies that a
// CloudFront-scoped web ACL returns an empty association list (CloudFront ACLs
// attach through distributions, not the regional ListResourcesForWebACL API),
// without reaching the WAFv2 API.
func TestWafAssociatedLoadBalancersEmptyForCloudFront(t *testing.T) {
	acl := &mqlAwsWafAcl{}
	acl.Scope = plugin.TValue[string]{Data: "CLOUDFRONT", State: plugin.StateIsSet}
	result, err := acl.associatedLoadBalancers()
	require.NoError(t, err)
	assert.Empty(t, result)
}
