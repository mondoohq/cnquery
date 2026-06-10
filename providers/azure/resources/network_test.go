// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/stretchr/testify/assert"
)

func TestFrontendIpConfigFields(t *testing.T) {
	t.Run("nil config returns empty strings", func(t *testing.T) {
		id, name, subnetId, publicIpId, privIp, alloc := frontendIpConfigFields(nil)
		assert.Empty(t, id+name+subnetId+publicIpId+privIp+alloc)
	})
	t.Run("internal (private) frontend has subnet, no public IP", func(t *testing.T) {
		id, name := "/fe/id", "privateFrontend"
		subnet := "/subnets/internal"
		priv, alloc := "10.0.0.4", network.IPAllocationMethodStatic
		fc := &network.ApplicationGatewayFrontendIPConfiguration{
			ID:   &id,
			Name: &name,
			Properties: &network.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
				Subnet:                    &network.SubResource{ID: &subnet},
				PrivateIPAddress:          &priv,
				PrivateIPAllocationMethod: &alloc,
			},
		}
		gotId, gotName, gotSubnet, gotPublic, gotPriv, gotAlloc := frontendIpConfigFields(fc)
		assert.Equal(t, "/fe/id", gotId)
		assert.Equal(t, "privateFrontend", gotName)
		assert.Equal(t, "/subnets/internal", gotSubnet)
		assert.Empty(t, gotPublic)
		assert.Equal(t, "10.0.0.4", gotPriv)
		assert.Equal(t, "Static", gotAlloc)
	})
	t.Run("internet-facing frontend has public IP, no subnet", func(t *testing.T) {
		pip := "/publicIPAddresses/pip"
		fc := &network.ApplicationGatewayFrontendIPConfiguration{
			Properties: &network.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
				PublicIPAddress: &network.SubResource{ID: &pip},
			},
		}
		_, _, gotSubnet, gotPublic, _, _ := frontendIpConfigFields(fc)
		assert.Empty(t, gotSubnet)
		assert.Equal(t, "/publicIPAddresses/pip", gotPublic)
	})
}

func TestParseAzurePortRange(t *testing.T) {
	entry := "*,80,1024-65535"
	ranges := parseAzureSecurityRulePortRange(entry)
	assert.Equal(t, 3, len(ranges))
	assert.Equal(t, "*", ranges[0].FromPort)
	assert.Equal(t, "*", ranges[0].ToPort)
	assert.Equal(t, "80", ranges[1].FromPort)
	assert.Equal(t, "80", ranges[1].ToPort)
	assert.Equal(t, "1024", ranges[2].FromPort)
	assert.Equal(t, "65535", ranges[2].ToPort)
}
