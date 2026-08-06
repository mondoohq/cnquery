// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func TestRouteIsDefault(t *testing.T) {
	assert.True(t, routeIsDefault("0.0.0.0/0", ""))
	assert.True(t, routeIsDefault("", "::/0"))
	assert.True(t, routeIsDefault("0.0.0.0/0", "::/0"))

	// A more specific destination is not a default route, even one that looks
	// close to the all-addresses form.
	assert.False(t, routeIsDefault("10.0.0.0/16", ""))
	assert.False(t, routeIsDefault("0.0.0.0/1", ""))
	assert.False(t, routeIsDefault("", "2600:1f18::/56"))
	assert.False(t, routeIsDefault("", ""))
}

func TestBlockModeStopsIngress(t *testing.T) {
	assert.True(t, blockModeStopsIngress(string(ec2types.InternetGatewayBlockModeBlockBidirectional)))
	assert.True(t, blockModeStopsIngress(string(ec2types.InternetGatewayBlockModeBlockIngress)))

	assert.False(t, blockModeStopsIngress(string(ec2types.InternetGatewayBlockModeOff)))
	// A subnet the API reported no mode for is not blocking anything.
	assert.False(t, blockModeStopsIngress(""))
}

// TestInternetGatewayIdPrefix pins the distinction isPublic depends on: an
// egress-only internet gateway carries outbound IPv6 traffic only, so its ID
// must not satisfy the internet gateway prefix check.
func TestInternetGatewayIdPrefix(t *testing.T) {
	assert.True(t, hasInternetGatewayPrefix("igw-0123456789abcdef0"))
	assert.False(t, hasInternetGatewayPrefix("eigw-0123456789abcdef0"))
	assert.False(t, hasInternetGatewayPrefix("vgw-0123456789abcdef0"))
	assert.False(t, hasInternetGatewayPrefix("local"))
	assert.False(t, hasInternetGatewayPrefix(""))
}
