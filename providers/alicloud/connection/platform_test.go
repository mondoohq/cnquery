// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAlicloudPlatforms verifies every catalog entry is retrievable by name and
// that each constructor produces a platform consistent with its catalog entry.
func TestAlicloudPlatforms(t *testing.T) {
	for _, pi := range Platforms {
		assert.Same(t, pi, PlatformByName(pi.Name), pi.Name)
	}
	assert.True(t, PlatformByName("alicloud").Consistent(NewAccountPlatform("1234567890")))
	assert.True(t, PlatformByName("alicloud-ack-cluster").Consistent(NewAckClusterPlatform("c-abc")))
	assert.True(t, PlatformByName("alicloud-alb-loadbalancer").Consistent(NewAlbPlatform("alb-abc")))
	assert.True(t, PlatformByName("alicloud-nlb-loadbalancer").Consistent(NewNlbPlatform("nlb-abc")))
	assert.True(t, PlatformByName("alicloud-vpc").Consistent(NewVpcPlatform("vpc-abc")))
	assert.True(t, PlatformByName("alicloud-waf-instance").Consistent(NewWafInstancePlatform("waf-abc")))
	assert.True(t, PlatformByName("alicloud-cloud-firewall").Consistent(NewCloudFirewallPlatform("1234567890")))
}

// TestAlicloudIdentifiers pins the stable platform-id formats used for asset
// deduplication; changing them would orphan previously-scanned assets.
func TestAlicloudIdentifiers(t *testing.T) {
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/alicloud/account/1234567890", NewAccountIdentifier("1234567890"))
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/alicloud/ack/cluster/c-abc", NewAckClusterIdentifier("c-abc"))
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/alicloud/alb/loadbalancer/alb-abc", NewAlbIdentifier("alb-abc"))
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/alicloud/nlb/loadbalancer/nlb-abc", NewNlbIdentifier("nlb-abc"))
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/alicloud/vpc/network/vpc-abc", NewVpcIdentifier("vpc-abc"))
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/alicloud/waf/instance/waf-abc", NewWafInstanceIdentifier("waf-abc"))
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/alicloud/cloudfirewall/1234567890", NewCloudFirewallIdentifier("1234567890"))
}
