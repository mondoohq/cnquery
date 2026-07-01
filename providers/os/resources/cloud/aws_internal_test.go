// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMacDetails_Unmarshal_Shapes verifies that the IMDS list/scalar fields
// deserialize across every shape the metadata service / crawler can produce,
// rather than erroring when an interface has more than one security group, more
// than one VPC CIDR block, or a non-numeric owner-id.
// Regression test for the cloud.instance failure on multi-valued ENIs.
func TestMacDetails_Unmarshal_Shapes(t *testing.T) {
	tests := []struct {
		name             string
		json             string
		wantSecGroupIDs  []string
		wantVPCCIDRs     []string
		wantPublicIPv4s  []string
		wantOwnerID      string
		wantPublicIPFrom string
	}{
		{
			name: "single value (account-owned, one SG) — the working control case",
			json: `{
				"owner-id": 339713021473,
				"security-group-ids": "sg-0756e626db22be220",
				"vpc-ipv4-cidr-blocks": "10.0.0.0/16",
				"public-ipv4s": "3.122.56.61",
				"local-ipv4s": "10.0.101.220",
				"subnet-ipv4-cidr-block": "10.0.101.0/24"
			}`,
			wantSecGroupIDs:  []string{"sg-0756e626db22be220"},
			wantVPCCIDRs:     []string{"10.0.0.0/16"},
			wantPublicIPv4s:  []string{"3.122.56.61"},
			wantOwnerID:      "339713021473",
			wantPublicIPFrom: "3.122.56.61",
		},
		{
			name: "multiple security groups + CIDRs as arrays",
			json: `{
				"owner-id": 339713021473,
				"security-group-ids": ["sg-aaaaaaaa", "sg-bbbbbbbb"],
				"vpc-ipv4-cidr-blocks": ["10.0.0.0/16", "10.1.0.0/16"],
				"public-ipv4s": "3.122.56.61"
			}`,
			wantSecGroupIDs:  []string{"sg-aaaaaaaa", "sg-bbbbbbbb"},
			wantVPCCIDRs:     []string{"10.0.0.0/16", "10.1.0.0/16"},
			wantPublicIPv4s:  []string{"3.122.56.61"},
			wantOwnerID:      "339713021473",
			wantPublicIPFrom: "3.122.56.61",
		},
		{
			name: "multiple security groups rendered as an object (legacy crawler)",
			json: `{
				"security-group-ids": {"sg-aaaaaaaa": {}, "sg-bbbbbbbb": {}}
			}`,
			wantSecGroupIDs: []string{"sg-aaaaaaaa", "sg-bbbbbbbb"},
		},
		{
			name: "service-managed ENI with non-numeric owner-id",
			json: `{
				"owner-id": "amazon-elb",
				"security-group-ids": "sg-cccccccc"
			}`,
			wantSecGroupIDs: []string{"sg-cccccccc"},
			wantOwnerID:     "amazon-elb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var md MacDetails
			require.NoError(t, json.Unmarshal([]byte(tt.json), &md),
				"MacDetails must deserialize every IMDS shape without error")

			if tt.wantSecGroupIDs != nil {
				assert.Equal(t, tt.wantSecGroupIDs, []string(md.SecurityGroupIDs))
			}
			if tt.wantVPCCIDRs != nil {
				assert.Equal(t, tt.wantVPCCIDRs, []string(md.VPCIPv4CIDRBlocks))
			}
			if tt.wantPublicIPv4s != nil {
				assert.Equal(t, tt.wantPublicIPv4s, []string(md.PublicIPv4s))
			}
			if tt.wantOwnerID != "" {
				assert.Equal(t, tt.wantOwnerID, string(md.OwnerID))
			}
			if tt.wantPublicIPFrom != "" {
				ip, ok := md.PublicIP()
				assert.True(t, ok)
				assert.Equal(t, tt.wantPublicIPFrom, ip.IP)
			}
		})
	}
}

// TestAWSNetwork_Unmarshal_MultiSG ensures the full network blob (the shape
// aws.Instance() unmarshals) parses when an interface carries multiple security
// groups, and that the public/private IPs are still extracted.
func TestAWSNetwork_Unmarshal_MultiSG(t *testing.T) {
	const blob = `{
		"interfaces": {
			"macs": {
				"02:90:b6:37:7b:fb": {
					"owner-id": 339713021473,
					"security-group-ids": ["sg-aaaaaaaa", "sg-bbbbbbbb"],
					"public-ipv4s": "3.122.56.61",
					"local-ipv4s": "10.0.101.220",
					"subnet-ipv4-cidr-block": "10.0.101.0/24",
					"vpc-ipv4-cidr-blocks": ["10.0.0.0/16", "10.1.0.0/16"]
				}
			}
		}
	}`

	var network AWSNetwork
	require.NoError(t, json.Unmarshal([]byte(blob), &network))

	md, ok := network.Interfaces.Macs["02:90:b6:37:7b:fb"]
	require.True(t, ok)
	assert.Equal(t, []string{"sg-aaaaaaaa", "sg-bbbbbbbb"}, []string(md.SecurityGroupIDs))

	pub, ok := md.PublicIP()
	assert.True(t, ok)
	assert.Equal(t, "3.122.56.61", pub.IP)

	priv, ok := md.PrivateIP()
	assert.True(t, ok)
	assert.Equal(t, "10.0.101.220", priv.IP)
}
