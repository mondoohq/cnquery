// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	netcommon "github.com/nutanix/ntnx-api-golang-clients/networking-go-client/v4/models/common/v1/config"
	netconfig "github.com/nutanix/ntnx-api-golang-clients/networking-go-client/v4/models/networking/v4/config"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/nutanix/connection"
	"go.mondoo.com/mql/v13/types"
)

// ---------------------------------------------------------------------------
// networking IP helpers
// ---------------------------------------------------------------------------

func netIPToString(ip *netcommon.IPAddress) string {
	if ip == nil {
		return ""
	}
	if ip.Ipv4 != nil && ip.Ipv4.Value != nil {
		return *ip.Ipv4.Value
	}
	if ip.Ipv6 != nil && ip.Ipv6.Value != nil {
		return *ip.Ipv6.Value
	}
	return ""
}

// ipSubnetToString renders an IPSubnet as a CIDR string ("10.0.0.0/24").
func ipSubnetToString(s *netconfig.IPSubnet) string {
	if s == nil {
		return ""
	}
	if s.Ipv4 != nil && s.Ipv4.Ip != nil && s.Ipv4.Ip.Value != nil {
		return fmt.Sprintf("%s/%d", *s.Ipv4.Ip.Value, derefInt(s.Ipv4.PrefixLength))
	}
	if s.Ipv6 != nil && s.Ipv6.Ip != nil && s.Ipv6.Ip.Value != nil {
		return fmt.Sprintf("%s/%d", *s.Ipv6.Ip.Value, derefInt(s.Ipv6.PrefixLength))
	}
	return ""
}

// ---------------------------------------------------------------------------
// VPCs
// ---------------------------------------------------------------------------

func newMqlVpc(runtime *plugin.Runtime, v *netconfig.Vpc) (*mqlNutanixNetworkVpc, error) {
	vpcType := ""
	if v.VpcType != nil {
		vpcType = v.VpcType.GetName()
	}
	prefixes := []any{}
	for i := range v.ExternallyRoutablePrefixes {
		prefixes = append(prefixes, ipSubnetToString(&v.ExternallyRoutablePrefixes[i]))
	}
	snatIps := []any{}
	for i := range v.SnatIps {
		snatIps = append(snatIps, netIPToString(&v.SnatIps[i]))
	}
	res, err := CreateResource(runtime, "nutanix.network.vpc", map[string]*llx.RawData{
		"__id":                           llx.StringDataPtr(v.ExtId),
		"id":                             llx.StringDataPtr(v.ExtId),
		"name":                           llx.StringDataPtr(v.Name),
		"description":                    llx.StringDataPtr(v.Description),
		"vpcType":                        llx.StringData(vpcType),
		"externallyRoutablePrefixes":     llx.ArrayData(prefixes, types.String),
		"snatIps":                        llx.ArrayData(snatIps, types.String),
		"externalRoutingDomainReference": llx.StringDataPtr(v.ExternalRoutingDomainReference),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlNutanixNetworkVpc), nil
}

func (a *mqlNutanix) vpcs() ([]any, error) {
	conn := a.conn()
	api := conn.VpcsApi()
	limit := pageSize
	res := []any{}
	for page := 0; ; page++ {
		p := page
		resp, err := guard(conn.NetMu(), func() (*netconfig.ListVpcsApiResponse, error) {
			return api.ListVpcs(&p, &limit, nil, nil, nil)
		})
		if err != nil {
			return nil, err
		}
		data := resp.GetData()
		if data == nil {
			break
		}
		items, ok := data.([]netconfig.Vpc)
		if !ok {
			return nil, fmt.Errorf("nutanix: unexpected response type %T from ListVpcs", data)
		}
		for i := range items {
			mqlVpc, err := newMqlVpc(a.MqlRuntime, &items[i])
			if err != nil {
				return nil, err
			}
			res = append(res, mqlVpc)
		}
		if len(items) < limit {
			break
		}
	}
	return res, nil
}

func vpcByID(runtime *plugin.Runtime, vpcID string) (*mqlNutanixNetworkVpc, error) {
	if v, ok := cachedResource[*mqlNutanixNetworkVpc](runtime, "nutanix.network.vpc", vpcID); ok {
		return v, nil
	}
	conn := runtime.Connection.(*connection.NutanixConnection)
	id := vpcID
	resp, err := guard(conn.NetMu(), func() (*netconfig.GetVpcApiResponse, error) {
		return conn.VpcsApi().GetVpcById(&id)
	})
	if err != nil {
		return nil, err
	}
	data := resp.GetData()
	if data == nil {
		return nil, nil
	}
	vpc, ok := data.(netconfig.Vpc)
	if !ok {
		return nil, nil
	}
	return newMqlVpc(runtime, &vpc)
}

// ---------------------------------------------------------------------------
// subnets
// ---------------------------------------------------------------------------

func newMqlSubnet(runtime *plugin.Runtime, s *netconfig.Subnet) (*mqlNutanixNetworkSubnet, error) {
	subnetType := ""
	if s.SubnetType != nil {
		subnetType = s.SubnetType.GetName()
	}
	numAssignedIps := int64(0)
	numFreeIps := int64(0)
	if s.IpUsage != nil {
		numAssignedIps = derefInt64(s.IpUsage.NumAssignedIPs)
		numFreeIps = derefInt64(s.IpUsage.NumFreeIPs)
	}
	reserved := []any{}
	for i := range s.ReservedIpAddresses {
		reserved = append(reserved, netIPToString(&s.ReservedIpAddresses[i]))
	}
	res, err := CreateResource(runtime, "nutanix.network.subnet", map[string]*llx.RawData{
		"__id":                 llx.StringDataPtr(s.ExtId),
		"id":                   llx.StringDataPtr(s.ExtId),
		"name":                 llx.StringDataPtr(s.Name),
		"description":          llx.StringDataPtr(s.Description),
		"subnetType":           llx.StringData(subnetType),
		"networkId":            llx.IntData(derefInt(s.NetworkId)),
		"ipPrefix":             llx.StringDataPtr(s.IpPrefix),
		"hypervisorType":       llx.StringDataPtr(s.HypervisorType),
		"bridgeName":           llx.StringDataPtr(s.BridgeName),
		"isExternal":           llx.BoolData(derefBool(s.IsExternal)),
		"isNatEnabled":         llx.BoolData(derefBool(s.IsNatEnabled)),
		"isAdvancedNetworking": llx.BoolData(derefBool(s.IsAdvancedNetworking)),
		"numAssignedIps":       llx.IntData(numAssignedIps),
		"numFreeIps":           llx.IntData(numFreeIps),
		"reservedIpAddresses":  llx.ArrayData(reserved, types.String),
	})
	if err != nil {
		return nil, err
	}
	mqlSubnet := res.(*mqlNutanixNetworkSubnet)
	if s.ClusterReference != nil {
		mqlSubnet.cacheClusterId = *s.ClusterReference
	}
	if s.VpcReference != nil {
		mqlSubnet.cacheVpcId = *s.VpcReference
	}
	return mqlSubnet, nil
}

func (a *mqlNutanix) subnets() ([]any, error) {
	conn := a.conn()
	scopeCluster := conn.ClusterID()
	api := conn.SubnetsApi()
	limit := pageSize
	res := []any{}
	for page := 0; ; page++ {
		p := page
		resp, err := guard(conn.NetMu(), func() (*netconfig.ListSubnetsApiResponse, error) {
			return api.ListSubnets(&p, &limit, nil, nil, nil, nil)
		})
		if err != nil {
			return nil, err
		}
		data := resp.GetData()
		if data == nil {
			break
		}
		items, ok := data.([]netconfig.Subnet)
		if !ok {
			return nil, fmt.Errorf("nutanix: unexpected response type %T from ListSubnets", data)
		}
		for i := range items {
			s := items[i]
			if scopeCluster != "" && (s.ClusterReference == nil || *s.ClusterReference != scopeCluster) {
				continue
			}
			mqlSubnet, err := newMqlSubnet(a.MqlRuntime, &s)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlSubnet)
		}
		if len(items) < limit {
			break
		}
	}
	return res, nil
}

type mqlNutanixNetworkSubnetInternal struct {
	cacheClusterId string
	cacheVpcId     string
}

func (a *mqlNutanixNetworkSubnet) cluster() (*mqlNutanixCluster, error) {
	if a.cacheClusterId == "" {
		a.Cluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := clusterByID(a.MqlRuntime, a.cacheClusterId)
	if err != nil {
		return nil, err
	}
	if res == nil {
		a.Cluster.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return res, nil
}

func (a *mqlNutanixNetworkSubnet) vpc() (*mqlNutanixNetworkVpc, error) {
	if a.cacheVpcId == "" {
		a.Vpc.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := vpcByID(a.MqlRuntime, a.cacheVpcId)
	if err != nil {
		return nil, err
	}
	if res == nil {
		a.Vpc.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return res, nil
}

func subnetByID(runtime *plugin.Runtime, subnetID string) (*mqlNutanixNetworkSubnet, error) {
	if s, ok := cachedResource[*mqlNutanixNetworkSubnet](runtime, "nutanix.network.subnet", subnetID); ok {
		return s, nil
	}
	conn := runtime.Connection.(*connection.NutanixConnection)
	id := subnetID
	resp, err := guard(conn.NetMu(), func() (*netconfig.GetSubnetApiResponse, error) {
		return conn.SubnetsApi().GetSubnetById(&id)
	})
	if err != nil {
		return nil, err
	}
	data := resp.GetData()
	if data == nil {
		return nil, nil
	}
	subnet, ok := data.(netconfig.Subnet)
	if !ok {
		return nil, nil
	}
	return newMqlSubnet(runtime, &subnet)
}

// ---------------------------------------------------------------------------
// floating IPs
// ---------------------------------------------------------------------------

func (a *mqlNutanix) floatingIps() ([]any, error) {
	conn := a.conn()
	api := conn.FloatingIpsApi()
	limit := pageSize
	res := []any{}
	for page := 0; ; page++ {
		p := page
		resp, err := guard(conn.NetMu(), func() (*netconfig.ListFloatingIpsApiResponse, error) {
			return api.ListFloatingIps(&p, &limit, nil, nil, nil)
		})
		if err != nil {
			return nil, err
		}
		data := resp.GetData()
		if data == nil {
			break
		}
		items, ok := data.([]netconfig.FloatingIp)
		if !ok {
			return nil, fmt.Errorf("nutanix: unexpected response type %T from ListFloatingIps", data)
		}
		for i := range items {
			f := items[i]
			associationStatus := ""
			if f.AssociationStatus != nil {
				associationStatus = f.AssociationStatus.GetName()
			}
			mqlFip, err := CreateResource(a.MqlRuntime, "nutanix.network.floatingIp", map[string]*llx.RawData{
				"__id":                         llx.StringDataPtr(f.ExtId),
				"id":                           llx.StringDataPtr(f.ExtId),
				"name":                         llx.StringDataPtr(f.Name),
				"description":                  llx.StringDataPtr(f.Description),
				"floatingIpValue":              llx.StringDataPtr(f.FloatingIpValue),
				"privateIp":                    llx.StringDataPtr(f.PrivateIp),
				"associationStatus":            llx.StringData(associationStatus),
				"vmNicReference":               llx.StringDataPtr(f.VmNicReference),
				"loadBalancerSessionReference": llx.StringDataPtr(f.LoadBalancerSessionReference),
			})
			if err != nil {
				return nil, err
			}
			mf := mqlFip.(*mqlNutanixNetworkFloatingIp)
			if f.VpcReference != nil {
				mf.cacheVpcId = *f.VpcReference
			}
			if f.ExternalSubnetReference != nil {
				mf.cacheSubnetId = *f.ExternalSubnetReference
			}
			res = append(res, mf)
		}
		if len(items) < limit {
			break
		}
	}
	return res, nil
}

type mqlNutanixNetworkFloatingIpInternal struct {
	cacheVpcId    string
	cacheSubnetId string
}

func (a *mqlNutanixNetworkFloatingIp) vpc() (*mqlNutanixNetworkVpc, error) {
	if a.cacheVpcId == "" {
		a.Vpc.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := vpcByID(a.MqlRuntime, a.cacheVpcId)
	if err != nil {
		return nil, err
	}
	if res == nil {
		a.Vpc.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return res, nil
}

func (a *mqlNutanixNetworkFloatingIp) externalSubnet() (*mqlNutanixNetworkSubnet, error) {
	if a.cacheSubnetId == "" {
		a.ExternalSubnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := subnetByID(a.MqlRuntime, a.cacheSubnetId)
	if err != nil {
		return nil, err
	}
	if res == nil {
		a.ExternalSubnet.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return res, nil
}
