// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== natGateways =====

// A firewall with no NAT gateway mappings reports an empty list rather than
// null, so a policy can count the gateways without a null guard.
func TestFirewallNatGatewaysEmptyWhenNoMappings(t *testing.T) {
	f := &mqlAwsNetworkfirewallFirewall{}
	got, err := f.natGateways()
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.NotNil(t, got)
}

// The mapping list can carry a blank id, which would otherwise resolve to a
// gateway lookup that cannot succeed.
func TestFirewallNatGatewaysSkipsEmptyIds(t *testing.T) {
	f := &mqlAwsNetworkfirewallFirewall{}
	f.cacheNatGatewayIds = []string{"", ""}
	got, err := f.natGateways()
	require.NoError(t, err)
	assert.Empty(t, got)
}
