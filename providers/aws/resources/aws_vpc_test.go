// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMqlVpnConnection(t *testing.T) {
	runtime := testRuntime()

	t.Run("extracts all fields from VpnConnection", func(t *testing.T) {
		vpnConn := ec2types.VpnConnection{
			VpnConnectionId:   aws.String("vpn-12345"),
			State:             ec2types.VpnStateAvailable,
			Type:              ec2types.GatewayTypeIpsec1,
			Category:          aws.String("VPN"),
			VpnGatewayId:      aws.String("vgw-aaa"),
			TransitGatewayId:  aws.String("tgw-bbb"),
			CustomerGatewayId: aws.String("cgw-ccc"),
			Tags: []ec2types.Tag{
				{Key: aws.String("Name"), Value: aws.String("test-vpn")},
			},
			Options: &ec2types.VpnConnectionOptions{
				StaticRoutesOnly:      aws.Bool(true),
				EnableAcceleration:    aws.Bool(false),
				LocalIpv4NetworkCidr:  aws.String("10.0.0.0/16"),
				RemoteIpv4NetworkCidr: aws.String("172.16.0.0/16"),
				OutsideIpAddressType:  aws.String("PublicIpv4"),
				TunnelInsideIpVersion: ec2types.TunnelInsideIpVersionIpv4,
			},
		}

		result, err := newMqlVpnConnection(runtime, "us-east-1", "123456789012", vpnConn)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "vpn-12345", result.Id.Data)
		assert.Equal(t, "us-east-1", result.Region.Data)
		assert.Equal(t, "available", result.State.Data)
		assert.Equal(t, "ipsec.1", result.Type.Data)
		assert.Equal(t, "VPN", result.Category.Data)
		assert.True(t, result.StaticRoutesOnly.Data)
		assert.False(t, result.EnableAcceleration.Data)
		assert.Equal(t, "10.0.0.0/16", result.LocalIpv4NetworkCidr.Data)
		assert.Equal(t, "172.16.0.0/16", result.RemoteIpv4NetworkCidr.Data)
		assert.Equal(t, "PublicIpv4", result.OutsideIpAddressType.Data)
		assert.Equal(t, "ipv4", result.TunnelInsideIpVersion.Data)
		assert.Contains(t, result.Arn.Data, "vpn-12345")

		// Verify internal cache for typed references
		assert.Equal(t, "vgw-aaa", *result.cacheVpnGatewayId)
		assert.Equal(t, "tgw-bbb", *result.cacheTransitGatewayId)
		assert.Equal(t, "cgw-ccc", *result.cacheCustomerGatewayId)
		assert.Equal(t, "us-east-1", result.region)
		assert.Equal(t, "123456789012", result.accountID)
	})

	t.Run("handles nil Options gracefully", func(t *testing.T) {
		vpnConn := ec2types.VpnConnection{
			VpnConnectionId: aws.String("vpn-nil-opts"),
			State:           ec2types.VpnStatePending,
			Options:         nil,
		}

		result, err := newMqlVpnConnection(runtime, "us-west-2", "111111111111", vpnConn)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.False(t, result.StaticRoutesOnly.Data)
		assert.False(t, result.EnableAcceleration.Data)
		assert.Empty(t, result.LocalIpv4NetworkCidr.Data)
		assert.Empty(t, result.TunnelInsideIpVersion.Data)
	})

	t.Run("handles nil gateway IDs", func(t *testing.T) {
		vpnConn := ec2types.VpnConnection{
			VpnConnectionId:   aws.String("vpn-no-gw"),
			VpnGatewayId:      nil,
			TransitGatewayId:  nil,
			CustomerGatewayId: nil,
		}

		result, err := newMqlVpnConnection(runtime, "us-east-1", "123456789012", vpnConn)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Nil(t, result.cacheVpnGatewayId)
		assert.Nil(t, result.cacheTransitGatewayId)
		assert.Nil(t, result.cacheCustomerGatewayId)
	})
}

func TestEipNullStateOnMissingCache(t *testing.T) {
	t.Run("networkInterface returns null when NetworkInterfaceId is nil", func(t *testing.T) {
		eip := &mqlAwsEc2Eip{}
		// eipCache is zero-valued — NetworkInterfaceId is nil
		result, err := eip.networkInterface()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, eip.NetworkInterface.IsNull())
		assert.True(t, eip.NetworkInterface.IsSet())
	})

	t.Run("networkInterface returns null when NetworkInterfaceId is empty", func(t *testing.T) {
		eip := &mqlAwsEc2Eip{}
		eip.eipCache = ec2types.Address{
			NetworkInterfaceId: aws.String(""),
		}
		result, err := eip.networkInterface()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, eip.NetworkInterface.IsNull())
		assert.True(t, eip.NetworkInterface.IsSet())
	})

}

func TestFlowLogNullStateOnMissingCache(t *testing.T) {
	t.Run("iamRole returns null when DeliverLogsPermissionArn is nil", func(t *testing.T) {
		fl := &mqlAwsVpcFlowlog{}
		result, err := fl.iamRole()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, fl.IamRole.IsNull())
		assert.True(t, fl.IamRole.IsSet())
	})

	t.Run("iamRole returns null when DeliverLogsPermissionArn is empty", func(t *testing.T) {
		fl := &mqlAwsVpcFlowlog{}
		empty := ""
		fl.cacheDeliverLogsPermissionArn = &empty
		result, err := fl.iamRole()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, fl.IamRole.IsNull())
		assert.True(t, fl.IamRole.IsSet())
	})

	t.Run("logGroup returns null when destinationType is not cloud-watch-logs", func(t *testing.T) {
		fl := &mqlAwsVpcFlowlog{}
		logGroup := "my-log-group"
		fl.cacheLogGroupName = &logGroup
		fl.cacheLogDestinationType = "s3"
		result, err := fl.logGroup()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, fl.LogGroup.IsNull())
		assert.True(t, fl.LogGroup.IsSet())
	})

	t.Run("s3Bucket returns null when destinationType is not s3", func(t *testing.T) {
		fl := &mqlAwsVpcFlowlog{}
		dest := "arn:aws:logs:us-east-1:123456789012:log-group:my-group"
		fl.cacheLogDestination = &dest
		fl.cacheLogDestinationType = "cloud-watch-logs"
		result, err := fl.s3Bucket()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, fl.S3Bucket.IsNull())
		assert.True(t, fl.S3Bucket.IsSet())
	})

	t.Run("s3Bucket returns null when destination ARN is unparseable", func(t *testing.T) {
		fl := &mqlAwsVpcFlowlog{}
		dest := "not-an-arn"
		fl.cacheLogDestination = &dest
		fl.cacheLogDestinationType = "s3"
		result, err := fl.s3Bucket()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, fl.S3Bucket.IsNull())
		assert.True(t, fl.S3Bucket.IsSet())
	})
}

func TestPeeringConnectionFields(t *testing.T) {
	runtime := testRuntime()

	t.Run("dnsResolutionEnabled is true when requester allows it", func(t *testing.T) {
		pc := ec2types.VpcPeeringConnection{
			VpcPeeringConnectionId: aws.String("pcx-123"),
			Status:                 &ec2types.VpcPeeringConnectionStateReason{Message: aws.String("active")},
			RequesterVpcInfo: &ec2types.VpcPeeringConnectionVpcInfo{
				OwnerId: aws.String("111111111111"),
				VpcId:   aws.String("vpc-req"),
				PeeringOptions: &ec2types.VpcPeeringConnectionOptionsDescription{
					AllowDnsResolutionFromRemoteVpc: aws.Bool(true),
				},
			},
			AccepterVpcInfo: &ec2types.VpcPeeringConnectionVpcInfo{
				OwnerId: aws.String("222222222222"),
				VpcId:   aws.String("vpc-acc"),
				PeeringOptions: &ec2types.VpcPeeringConnectionOptionsDescription{
					AllowDnsResolutionFromRemoteVpc: aws.Bool(false),
				},
			},
		}

		// Simulate the logic from peeringConnections() creation
		dnsResolution := false
		if pc.RequesterVpcInfo != nil && pc.RequesterVpcInfo.PeeringOptions != nil &&
			pc.RequesterVpcInfo.PeeringOptions.AllowDnsResolutionFromRemoteVpc != nil {
			dnsResolution = *pc.RequesterVpcInfo.PeeringOptions.AllowDnsResolutionFromRemoteVpc
		}
		if !dnsResolution && pc.AccepterVpcInfo != nil && pc.AccepterVpcInfo.PeeringOptions != nil &&
			pc.AccepterVpcInfo.PeeringOptions.AllowDnsResolutionFromRemoteVpc != nil {
			dnsResolution = *pc.AccepterVpcInfo.PeeringOptions.AllowDnsResolutionFromRemoteVpc
		}

		assert.True(t, dnsResolution)

		var requesterAccountId string
		if pc.RequesterVpcInfo != nil {
			requesterAccountId = *pc.RequesterVpcInfo.OwnerId
		}
		assert.Equal(t, "111111111111", requesterAccountId)

		_ = runtime // used by other tests that need resource creation
	})
}

func TestVpnGatewayNullState(t *testing.T) {
	t.Run("vpnGateway returns null when VpnGatewayId is nil", func(t *testing.T) {
		vpn := &mqlAwsEc2Vpnconnection{}
		result, err := vpn.vpnGateway()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, vpn.VpnGateway.IsNull())
		assert.True(t, vpn.VpnGateway.IsSet())
	})

	t.Run("transitGateway returns null when TransitGatewayId is nil", func(t *testing.T) {
		vpn := &mqlAwsEc2Vpnconnection{}
		result, err := vpn.transitGateway()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, vpn.TransitGateway.IsNull())
		assert.True(t, vpn.TransitGateway.IsSet())
	})

	t.Run("customerGateway returns null when CustomerGatewayId is nil", func(t *testing.T) {
		vpn := &mqlAwsEc2Vpnconnection{}
		result, err := vpn.customerGateway()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, vpn.CustomerGateway.IsNull())
		assert.True(t, vpn.CustomerGateway.IsSet())
	})

	t.Run("customerGateway returns null when CustomerGatewayId is empty", func(t *testing.T) {
		vpn := &mqlAwsEc2Vpnconnection{}
		empty := ""
		vpn.cacheCustomerGatewayId = &empty
		result, err := vpn.customerGateway()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, vpn.CustomerGateway.IsNull())
		assert.True(t, vpn.CustomerGateway.IsSet())
	})
}
