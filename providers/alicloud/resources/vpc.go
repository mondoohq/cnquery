// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"time"

	vpcclient "github.com/alibabacloud-go/vpc-20160428/v6/client"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
	"go.mondoo.com/mql/v13/types"
)

// vpcParseTime parses an RFC3339 timestamp string returned by the Alibaba Cloud
// APIs. It returns nil on a nil pointer or an unparseable value so the field
// resolves to null rather than a zero time.
func vpcParseTime(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}

// vpcStrSlice converts a slice of *string into a []any of non-nil string
// values, safe to pass to llx.ArrayData with types.String.
func vpcStrSlice(in []*string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		if s != nil {
			out = append(out, *s)
		}
	}
	return out
}

// vpcTagMap builds a string map from the common Alibaba Cloud Key/Value tag
// pair shape.
func vpcTagMap(pairs []*vpcTag) map[string]any {
	out := map[string]any{}
	for _, p := range pairs {
		if p == nil || p.key == nil {
			continue
		}
		v := ""
		if p.value != nil {
			v = *p.value
		}
		out[*p.key] = v
	}
	return out
}

// vpcTag is a normalized tag pair used by vpcTagMap so the various SDK
// tag shapes can be flattened through one helper.
type vpcTag struct {
	key   *string
	value *string
}

// vpcStr dereferences a *string, returning "" when nil. Used for dict
// values, which carry plain Go scalars rather than RawData.
func vpcStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// vpcInt32 dereferences a *int32, returning 0 when nil.
func vpcInt32(v *int32) int64 {
	if v == nil {
		return 0
	}
	return int64(*v)
}

// vpcBool dereferences a *bool, returning false when nil.
func vpcBool(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

// vpcAclTotalCount parses the DescribeNetworkAcls TotalCount, which the SDK
// models as a string. It returns -1 when the value is nil or unparseable so the
// caller stops paginating.
func vpcAclTotalCount(s *string) int64 {
	if s == nil || *s == "" {
		return -1
	}
	n, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		return -1
	}
	return n
}

func (r *mqlAlicloudVpc) id() (string, error) {
	return "alicloud.vpc", nil
}

func (r *mqlAlicloudVpc) networks() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.VpcClient(region)
		if err != nil {
			return nil, err
		}

		pageNumber := int32(1)
		pageSize := int32(50)
		for {
			req := &vpcclient.DescribeVpcsRequest{
				RegionId:   &region,
				PageNumber: &pageNumber,
				PageSize:   &pageSize,
			}
			resp, err := client.DescribeVpcs(req)
			if err != nil {
				return nil, err
			}
			if resp == nil || resp.Body == nil || resp.Body.Vpcs == nil {
				break
			}

			for _, vpc := range resp.Body.Vpcs.Vpc {
				if vpc == nil || vpc.VpcId == nil {
					continue
				}

				secondary := []any{}
				if vpc.SecondaryCidrBlocks != nil {
					secondary = vpcStrSlice(vpc.SecondaryCidrBlocks.SecondaryCidrBlock)
				}
				userCidrs := []any{}
				if vpc.UserCidrs != nil {
					userCidrs = vpcStrSlice(vpc.UserCidrs.UserCidr)
				}
				vswitchIds := []any{}
				if vpc.VSwitchIds != nil {
					vswitchIds = vpcStrSlice(vpc.VSwitchIds.VSwitchId)
				}
				natGatewayIds := []any{}
				if vpc.NatGatewayIds != nil {
					natGatewayIds = vpcStrSlice(vpc.NatGatewayIds.NatGatewayIds)
				}
				routeTableIds := []any{}
				if vpc.RouterTableIds != nil {
					routeTableIds = vpcStrSlice(vpc.RouterTableIds.RouterTableIds)
				}

				ipv6Blocks := []any{}
				if vpc.Ipv6CidrBlocks != nil {
					for _, b := range vpc.Ipv6CidrBlocks.Ipv6CidrBlock {
						if b == nil {
							continue
						}
						ipv6Blocks = append(ipv6Blocks, map[string]any{
							"ipv6CidrBlock": vpcStr(b.Ipv6CidrBlock),
							"ipv6Isp":       vpcStr(b.Ipv6Isp),
						})
					}
				}

				tags := map[string]any{}
				if vpc.Tags != nil {
					pairs := make([]*vpcTag, 0, len(vpc.Tags.Tag))
					for _, t := range vpc.Tags.Tag {
						if t != nil {
							pairs = append(pairs, &vpcTag{key: t.Key, value: t.Value})
						}
					}
					tags = vpcTagMap(pairs)
				}

				network, err := CreateResource(r.MqlRuntime, "alicloud.vpc.network", map[string]*llx.RawData{
					"__id":                 llx.StringDataPtr(vpc.VpcId),
					"vpcId":                llx.StringDataPtr(vpc.VpcId),
					"vpcName":              llx.StringDataPtr(vpc.VpcName),
					"description":          llx.StringDataPtr(vpc.Description),
					"cidrBlock":            llx.StringDataPtr(vpc.CidrBlock),
					"cidrBlocks":           llx.ArrayData(secondary, types.String),
					"ipv6CidrBlock":        llx.StringDataPtr(vpc.Ipv6CidrBlock),
					"ipv6CidrBlocks":       llx.ArrayData(ipv6Blocks, types.Dict),
					"enabledIpv6":          llx.BoolDataPtr(vpc.EnabledIpv6),
					"status":               llx.StringDataPtr(vpc.Status),
					"isDefault":            llx.BoolDataPtr(vpc.IsDefault),
					"regionId":             llx.StringDataPtr(vpc.RegionId),
					"vRouterId":            llx.StringDataPtr(vpc.VRouterId),
					"creationTime":         llx.TimeDataPtr(vpcParseTime(vpc.CreationTime)),
					"resourceGroupId":      llx.StringDataPtr(vpc.ResourceGroupId),
					"userCidrs":            llx.ArrayData(userCidrs, types.String),
					"vswitchIds":           llx.ArrayData(vswitchIds, types.String),
					"natGatewayIds":        llx.ArrayData(natGatewayIds, types.String),
					"routeTableIds":        llx.ArrayData(routeTableIds, types.String),
					"dnsHostnameStatus":    llx.StringDataPtr(vpc.DnsHostnameStatus),
					"dhcpOptionsSetId":     llx.StringDataPtr(vpc.DhcpOptionsSetId),
					"dhcpOptionsSetStatus": llx.StringDataPtr(vpc.DhcpOptionsSetStatus),
					"cenStatus":            llx.StringDataPtr(vpc.CenStatus),
					"ownerId":              llx.IntDataPtr(vpc.OwnerId),
					"tags":                 llx.MapData(tags, types.String),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, network)
			}

			if resp.Body.TotalCount == nil || int64(pageNumber)*int64(pageSize) >= int64(*resp.Body.TotalCount) {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

func (r *mqlAlicloudVpcNetwork) id() (string, error) {
	return r.VpcId.Data, nil
}

func (r *mqlAlicloudVpc) vswitches() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.VpcClient(region)
		if err != nil {
			return nil, err
		}

		pageNumber := int32(1)
		pageSize := int32(50)
		for {
			req := &vpcclient.DescribeVSwitchesRequest{
				RegionId:   &region,
				PageNumber: &pageNumber,
				PageSize:   &pageSize,
			}
			resp, err := client.DescribeVSwitches(req)
			if err != nil {
				return nil, err
			}
			if resp == nil || resp.Body == nil || resp.Body.VSwitches == nil {
				break
			}

			for _, vsw := range resp.Body.VSwitches.VSwitch {
				if vsw == nil || vsw.VSwitchId == nil {
					continue
				}

				var routeTableId, routeTableType *string
				if vsw.RouteTable != nil {
					routeTableId = vsw.RouteTable.RouteTableId
					routeTableType = vsw.RouteTable.RouteTableType
				}

				tags := map[string]any{}
				if vsw.Tags != nil {
					pairs := make([]*vpcTag, 0, len(vsw.Tags.Tag))
					for _, t := range vsw.Tags.Tag {
						if t != nil {
							pairs = append(pairs, &vpcTag{key: t.Key, value: t.Value})
						}
					}
					tags = vpcTagMap(pairs)
				}

				vswitch, err := CreateResource(r.MqlRuntime, "alicloud.vpc.vswitch", map[string]*llx.RawData{
					"__id":                    llx.StringDataPtr(vsw.VSwitchId),
					"vSwitchId":               llx.StringDataPtr(vsw.VSwitchId),
					"vSwitchName":             llx.StringDataPtr(vsw.VSwitchName),
					"description":             llx.StringDataPtr(vsw.Description),
					"cidrBlock":               llx.StringDataPtr(vsw.CidrBlock),
					"ipv6CidrBlock":           llx.StringDataPtr(vsw.Ipv6CidrBlock),
					"enabledIpv6":             llx.BoolDataPtr(vsw.EnabledIpv6),
					"status":                  llx.StringDataPtr(vsw.Status),
					"vpcId":                   llx.StringDataPtr(vsw.VpcId),
					"zoneId":                  llx.StringDataPtr(vsw.ZoneId),
					"availableIpAddressCount": llx.IntDataPtr(vsw.AvailableIpAddressCount),
					"isDefault":               llx.BoolDataPtr(vsw.IsDefault),
					"creationTime":            llx.TimeDataPtr(vpcParseTime(vsw.CreationTime)),
					"routeTableId":            llx.StringDataPtr(routeTableId),
					"routeTableType":          llx.StringDataPtr(routeTableType),
					"networkAclId":            llx.StringDataPtr(vsw.NetworkAclId),
					"resourceGroupId":         llx.StringDataPtr(vsw.ResourceGroupId),
					"shareType":               llx.StringDataPtr(vsw.ShareType),
					"ownerId":                 llx.IntDataPtr(vsw.OwnerId),
					"tags":                    llx.MapData(tags, types.String),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, vswitch)
			}

			if resp.Body.TotalCount == nil || int64(pageNumber)*int64(pageSize) >= int64(*resp.Body.TotalCount) {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

func (r *mqlAlicloudVpcVswitch) id() (string, error) {
	return r.VSwitchId.Data, nil
}

func (r *mqlAlicloudVpc) routeTables() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.VpcClient(region)
		if err != nil {
			return nil, err
		}

		pageNumber := int32(1)
		pageSize := int32(50)
		for {
			req := &vpcclient.DescribeRouteTableListRequest{
				RegionId:   &region,
				PageNumber: &pageNumber,
				PageSize:   &pageSize,
			}
			resp, err := client.DescribeRouteTableList(req)
			if err != nil {
				return nil, err
			}
			if resp == nil || resp.Body == nil || resp.Body.RouterTableList == nil {
				break
			}

			for _, rt := range resp.Body.RouterTableList.RouterTableListType {
				if rt == nil || rt.RouteTableId == nil {
					continue
				}

				vSwitchIds := []any{}
				if rt.VSwitchIds != nil {
					vSwitchIds = vpcStrSlice(rt.VSwitchIds.VSwitchId)
				}
				gatewayIds := []any{}
				if rt.GatewayIds != nil {
					gatewayIds = vpcStrSlice(rt.GatewayIds.GatewayIds)
				}

				tags := map[string]any{}
				if rt.Tags != nil {
					pairs := make([]*vpcTag, 0, len(rt.Tags.Tag))
					for _, t := range rt.Tags.Tag {
						if t != nil {
							pairs = append(pairs, &vpcTag{key: t.Key, value: t.Value})
						}
					}
					tags = vpcTagMap(pairs)
				}

				routeTable, err := CreateResource(r.MqlRuntime, "alicloud.vpc.routeTable", map[string]*llx.RawData{
					"__id":                   llx.StringDataPtr(rt.RouteTableId),
					"routeTableId":           llx.StringDataPtr(rt.RouteTableId),
					"routeTableName":         llx.StringDataPtr(rt.RouteTableName),
					"routeTableType":         llx.StringDataPtr(rt.RouteTableType),
					"description":            llx.StringDataPtr(rt.Description),
					"vpcId":                  llx.StringDataPtr(rt.VpcId),
					"status":                 llx.StringDataPtr(rt.Status),
					"creationTime":           llx.TimeDataPtr(vpcParseTime(rt.CreationTime)),
					"resourceGroupId":        llx.StringDataPtr(rt.ResourceGroupId),
					"vSwitchIds":             llx.ArrayData(vSwitchIds, types.String),
					"gatewayIds":             llx.ArrayData(gatewayIds, types.String),
					"associateType":          llx.StringDataPtr(rt.AssociateType),
					"routerId":               llx.StringDataPtr(rt.RouterId),
					"routerType":             llx.StringDataPtr(rt.RouterType),
					"routePropagationEnable": llx.BoolDataPtr(rt.RoutePropagationEnable),
					"ownerId":                llx.IntDataPtr(rt.OwnerId),
					"tags":                   llx.MapData(tags, types.String),
				})
				if err != nil {
					return nil, err
				}
				mqlRt := routeTable.(*mqlAlicloudVpcRouteTable)
				mqlRt.region = region
				res = append(res, routeTable)
			}

			if resp.Body.TotalCount == nil || int64(pageNumber)*int64(pageSize) >= int64(*resp.Body.TotalCount) {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

// mqlAlicloudVpcRouteTableInternal caches the region the route table was
// discovered in so the route entries can be fetched from the correct regional
// endpoint.
type mqlAlicloudVpcRouteTableInternal struct {
	region string
}

func (r *mqlAlicloudVpcRouteTable) id() (string, error) {
	return r.RouteTableId.Data, nil
}

func (r *mqlAlicloudVpcRouteTable) routeEntries() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	region := r.region
	if region == "" {
		// The route table was not created through the regional fan-out, so the
		// region needed to reach the route-entry endpoint is unknown.
		return []any{}, nil
	}
	client, err := conn.VpcClient(region)
	if err != nil {
		return nil, err
	}

	res := []any{}
	var nextToken *string
	maxResult := int32(100)
	for {
		req := &vpcclient.DescribeRouteEntryListRequest{
			RegionId:     &region,
			RouteTableId: &r.RouteTableId.Data,
			MaxResult:    &maxResult,
			NextToken:    nextToken,
		}
		resp, err := client.DescribeRouteEntryList(req)
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil || resp.Body.RouteEntrys == nil {
			break
		}

		for _, entry := range resp.Body.RouteEntrys.RouteEntry {
			if entry == nil {
				continue
			}
			var nextHopType, nextHopId *string
			if entry.NextHops != nil {
				for _, h := range entry.NextHops.NextHop {
					if h != nil {
						nextHopType = h.NextHopType
						nextHopId = h.NextHopId
						break
					}
				}
			}
			res = append(res, map[string]any{
				"routeEntryId":         vpcStr(entry.RouteEntryId),
				"routeEntryName":       vpcStr(entry.RouteEntryName),
				"description":          vpcStr(entry.Description),
				"destinationCidrBlock": vpcStr(entry.DestinationCidrBlock),
				"nextHopType":          vpcStr(nextHopType),
				"nextHopId":            vpcStr(nextHopId),
				"status":               vpcStr(entry.Status),
				"type":                 vpcStr(entry.Type),
				"ipVersion":            vpcStr(entry.IpVersion),
				"origin":               vpcStr(entry.Origin),
			})
		}

		if resp.Body.NextToken == nil || *resp.Body.NextToken == "" {
			break
		}
		nextToken = resp.Body.NextToken
	}
	return res, nil
}

func (r *mqlAlicloudVpc) natGateways() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.VpcClient(region)
		if err != nil {
			return nil, err
		}

		pageNumber := int32(1)
		pageSize := int32(50)
		for {
			req := &vpcclient.DescribeNatGatewaysRequest{
				RegionId:   &region,
				PageNumber: &pageNumber,
				PageSize:   &pageSize,
			}
			resp, err := client.DescribeNatGateways(req)
			if err != nil {
				return nil, err
			}
			if resp == nil || resp.Body == nil || resp.Body.NatGateways == nil {
				break
			}

			for _, nat := range resp.Body.NatGateways.NatGateway {
				if nat == nil || nat.NatGatewayId == nil {
					continue
				}

				snatTableIds := []any{}
				if nat.SnatTableIds != nil {
					snatTableIds = vpcStrSlice(nat.SnatTableIds.SnatTableId)
				}
				forwardTableIds := []any{}
				if nat.ForwardTableIds != nil {
					forwardTableIds = vpcStrSlice(nat.ForwardTableIds.ForwardTableId)
				}
				fullNatTableIds := []any{}
				if nat.FullNatTableIds != nil {
					fullNatTableIds = vpcStrSlice(nat.FullNatTableIds.FullNatTableId)
				}

				var privateInfo any
				if nat.NatGatewayPrivateInfo != nil {
					pi := nat.NatGatewayPrivateInfo
					privateInfo = map[string]any{
						"vswitchId":               vpcStr(pi.VswitchId),
						"privateIpAddress":        vpcStr(pi.PrivateIpAddress),
						"eniInstanceId":           vpcStr(pi.EniInstanceId),
						"eniType":                 vpcStr(pi.EniType),
						"izNo":                    vpcStr(pi.IzNo),
						"maxBandwidth":            vpcInt32(pi.MaxBandwidth),
						"maxSessionEstablishRate": vpcInt32(pi.MaxSessionEstablishRate),
						"maxSessionQuota":         vpcInt32(pi.MaxSessionQuota),
					}
				}

				var accessMode any
				if nat.AccessMode != nil {
					accessMode = map[string]any{
						"modeValue":  vpcStr(nat.AccessMode.ModeValue),
						"tunnelType": vpcStr(nat.AccessMode.TunnelType),
					}
				}

				ipLists := []any{}
				if nat.IpLists != nil {
					for _, ip := range nat.IpLists.IpList {
						if ip == nil {
							continue
						}
						ipLists = append(ipLists, map[string]any{
							"allocationId":     vpcStr(ip.AllocationId),
							"ipAddress":        vpcStr(ip.IpAddress),
							"privateIpAddress": vpcStr(ip.PrivateIpAddress),
							"snatEntryEnabled": vpcBool(ip.SnatEntryEnabled),
							"usingStatus":      vpcStr(ip.UsingStatus),
						})
					}
				}

				tags := map[string]any{}
				if nat.Tags != nil {
					pairs := make([]*vpcTag, 0, len(nat.Tags.Tag))
					for _, t := range nat.Tags.Tag {
						if t != nil {
							pairs = append(pairs, &vpcTag{key: t.TagKey, value: t.TagValue})
						}
					}
					tags = vpcTagMap(pairs)
				}

				natGateway, err := CreateResource(r.MqlRuntime, "alicloud.vpc.natGateway", map[string]*llx.RawData{
					"__id":                      llx.StringDataPtr(nat.NatGatewayId),
					"natGatewayId":              llx.StringDataPtr(nat.NatGatewayId),
					"name":                      llx.StringDataPtr(nat.Name),
					"description":               llx.StringDataPtr(nat.Description),
					"vpcId":                     llx.StringDataPtr(nat.VpcId),
					"status":                    llx.StringDataPtr(nat.Status),
					"spec":                      llx.StringDataPtr(nat.Spec),
					"natType":                   llx.StringDataPtr(nat.NatType),
					"networkType":               llx.StringDataPtr(nat.NetworkType),
					"businessStatus":            llx.StringDataPtr(nat.BusinessStatus),
					"creationTime":              llx.TimeDataPtr(vpcParseTime(nat.CreationTime)),
					"expiredTime":               llx.TimeDataPtr(vpcParseTime(nat.ExpiredTime)),
					"internetChargeType":        llx.StringDataPtr(nat.InternetChargeType),
					"instanceChargeType":        llx.StringDataPtr(nat.InstanceChargeType),
					"deletionProtection":        llx.BoolDataPtr(nat.DeletionProtection),
					"ecsMetricEnabled":          llx.BoolDataPtr(nat.EcsMetricEnabled),
					"icmpReplyEnabled":          llx.BoolDataPtr(nat.IcmpReplyEnabled),
					"privateLinkEnabled":        llx.BoolDataPtr(nat.PrivateLinkEnabled),
					"privateLinkMode":           llx.StringDataPtr(nat.PrivateLinkMode),
					"securityProtectionEnabled": llx.BoolDataPtr(nat.SecurityProtectionEnabled),
					"eipBindMode":               llx.StringDataPtr(nat.EipBindMode),
					"enableSessionLog":          llx.StringDataPtr(nat.EnableSessionLog),
					"autoPay":                   llx.BoolDataPtr(nat.AutoPay),
					"regionId":                  llx.StringDataPtr(nat.RegionId),
					"resourceGroupId":           llx.StringDataPtr(nat.ResourceGroupId),
					"snatTableIds":              llx.ArrayData(snatTableIds, types.String),
					"forwardTableIds":           llx.ArrayData(forwardTableIds, types.String),
					"fullNatTableIds":           llx.ArrayData(fullNatTableIds, types.String),
					"natGatewayPrivateInfo":     llx.DictData(privateInfo),
					"ipLists":                   llx.ArrayData(ipLists, types.Dict),
					"accessMode":                llx.DictData(accessMode),
					"tags":                      llx.MapData(tags, types.String),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, natGateway)
			}

			if resp.Body.TotalCount == nil || int64(pageNumber)*int64(pageSize) >= int64(*resp.Body.TotalCount) {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

func (r *mqlAlicloudVpcNatGateway) id() (string, error) {
	return r.NatGatewayId.Data, nil
}

func (r *mqlAlicloudVpc) eipAddresses() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.VpcClient(region)
		if err != nil {
			return nil, err
		}

		pageNumber := int32(1)
		pageSize := int32(50)
		for {
			req := &vpcclient.DescribeEipAddressesRequest{
				RegionId:   &region,
				PageNumber: &pageNumber,
				PageSize:   &pageSize,
			}
			resp, err := client.DescribeEipAddresses(req)
			if err != nil {
				return nil, err
			}
			if resp == nil || resp.Body == nil || resp.Body.EipAddresses == nil {
				break
			}

			for _, eip := range resp.Body.EipAddresses.EipAddress {
				if eip == nil || eip.AllocationId == nil {
					continue
				}

				securityProtectionTypes := []any{}
				if eip.SecurityProtectionTypes != nil {
					securityProtectionTypes = vpcStrSlice(eip.SecurityProtectionTypes.SecurityProtectionType)
				}

				tags := map[string]any{}
				if eip.Tags != nil {
					pairs := make([]*vpcTag, 0, len(eip.Tags.Tag))
					for _, t := range eip.Tags.Tag {
						if t != nil {
							pairs = append(pairs, &vpcTag{key: t.Key, value: t.Value})
						}
					}
					tags = vpcTagMap(pairs)
				}

				eipAddress, err := CreateResource(r.MqlRuntime, "alicloud.vpc.eipAddress", map[string]*llx.RawData{
					"__id":                      llx.StringDataPtr(eip.AllocationId),
					"allocationId":              llx.StringDataPtr(eip.AllocationId),
					"name":                      llx.StringDataPtr(eip.Name),
					"description":               llx.StringDataPtr(eip.Description),
					"ipAddress":                 llx.StringDataPtr(eip.IpAddress),
					"status":                    llx.StringDataPtr(eip.Status),
					"bandwidth":                 llx.StringDataPtr(eip.Bandwidth),
					"internetChargeType":        llx.StringDataPtr(eip.InternetChargeType),
					"chargeType":                llx.StringDataPtr(eip.ChargeType),
					"isp":                       llx.StringDataPtr(eip.ISP),
					"instanceId":                llx.StringDataPtr(eip.InstanceId),
					"instanceType":              llx.StringDataPtr(eip.InstanceType),
					"instanceRegionId":          llx.StringDataPtr(eip.InstanceRegionId),
					"privateIpAddress":          llx.StringDataPtr(eip.PrivateIpAddress),
					"allocationTime":            llx.TimeDataPtr(vpcParseTime(eip.AllocationTime)),
					"expiredTime":               llx.TimeDataPtr(vpcParseTime(eip.ExpiredTime)),
					"hdMonitorStatus":           llx.StringDataPtr(eip.HDMonitorStatus),
					"deletionProtection":        llx.BoolDataPtr(eip.DeletionProtection),
					"hasReservationData":        llx.StringDataPtr(eip.HasReservationData),
					"netmode":                   llx.StringDataPtr(eip.Netmode),
					"publicIpAddressPoolId":     llx.StringDataPtr(eip.PublicIpAddressPoolId),
					"regionId":                  llx.StringDataPtr(eip.RegionId),
					"resourceGroupId":           llx.StringDataPtr(eip.ResourceGroupId),
					"vpcId":                     llx.StringDataPtr(eip.VpcId),
					"zone":                      llx.StringDataPtr(eip.Zone),
					"bandwidthPackageId":        llx.StringDataPtr(eip.BandwidthPackageId),
					"bandwidthPackageType":      llx.StringDataPtr(eip.BandwidthPackageType),
					"bandwidthPackageBandwidth": llx.StringDataPtr(eip.BandwidthPackageBandwidth),
					"eipBandwidth":              llx.StringDataPtr(eip.EipBandwidth),
					"bizType":                   llx.StringDataPtr(eip.BizType),
					"businessStatus":            llx.StringDataPtr(eip.BusinessStatus),
					"mode":                      llx.StringDataPtr(eip.Mode),
					"secondLimited":             llx.BoolDataPtr(eip.SecondLimited),
					"segmentInstanceId":         llx.StringDataPtr(eip.SegmentInstanceId),
					"securityProtectionTypes":   llx.ArrayData(securityProtectionTypes, types.String),
					"serviceManaged":            llx.IntDataPtr(eip.ServiceManaged),
					"tags":                      llx.MapData(tags, types.String),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, eipAddress)
			}

			if resp.Body.TotalCount == nil || int64(pageNumber)*int64(pageSize) >= int64(*resp.Body.TotalCount) {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

func (r *mqlAlicloudVpcEipAddress) id() (string, error) {
	return r.AllocationId.Data, nil
}

func (r *mqlAlicloudVpc) networkAcls() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.VpcClient(region)
		if err != nil {
			return nil, err
		}

		pageNumber := int32(1)
		pageSize := int32(50)
		for {
			req := &vpcclient.DescribeNetworkAclsRequest{
				RegionId:   &region,
				PageNumber: &pageNumber,
				PageSize:   &pageSize,
			}
			resp, err := client.DescribeNetworkAcls(req)
			if err != nil {
				return nil, err
			}
			if resp == nil || resp.Body == nil || resp.Body.NetworkAcls == nil {
				break
			}

			for _, acl := range resp.Body.NetworkAcls.NetworkAcl {
				if acl == nil || acl.NetworkAclId == nil {
					continue
				}

				ingress := []any{}
				if acl.IngressAclEntries != nil {
					for _, e := range acl.IngressAclEntries.IngressAclEntry {
						if e == nil {
							continue
						}
						ingress = append(ingress, map[string]any{
							"networkAclEntryId":   vpcStr(e.NetworkAclEntryId),
							"networkAclEntryName": vpcStr(e.NetworkAclEntryName),
							"description":         vpcStr(e.Description),
							"protocol":            vpcStr(e.Protocol),
							"port":                vpcStr(e.Port),
							"sourceCidrIp":        vpcStr(e.SourceCidrIp),
							"policy":              vpcStr(e.Policy),
							"entryType":           vpcStr(e.EntryType),
							"ipVersion":           vpcStr(e.IpVersion),
						})
					}
				}

				egress := []any{}
				if acl.EgressAclEntries != nil {
					for _, e := range acl.EgressAclEntries.EgressAclEntry {
						if e == nil {
							continue
						}
						egress = append(egress, map[string]any{
							"networkAclEntryId":   vpcStr(e.NetworkAclEntryId),
							"networkAclEntryName": vpcStr(e.NetworkAclEntryName),
							"description":         vpcStr(e.Description),
							"protocol":            vpcStr(e.Protocol),
							"port":                vpcStr(e.Port),
							"destinationCidrIp":   vpcStr(e.DestinationCidrIp),
							"policy":              vpcStr(e.Policy),
							"entryType":           vpcStr(e.EntryType),
							"ipVersion":           vpcStr(e.IpVersion),
						})
					}
				}

				resources := []any{}
				if acl.Resources != nil {
					for _, rsc := range acl.Resources.Resource {
						if rsc == nil {
							continue
						}
						resources = append(resources, map[string]any{
							"resourceId":   vpcStr(rsc.ResourceId),
							"resourceType": vpcStr(rsc.ResourceType),
							"status":       vpcStr(rsc.Status),
						})
					}
				}

				tags := map[string]any{}
				if acl.Tags != nil {
					pairs := make([]*vpcTag, 0, len(acl.Tags.Tag))
					for _, t := range acl.Tags.Tag {
						if t != nil {
							pairs = append(pairs, &vpcTag{key: t.Key, value: t.Value})
						}
					}
					tags = vpcTagMap(pairs)
				}

				networkAcl, err := CreateResource(r.MqlRuntime, "alicloud.vpc.networkAcl", map[string]*llx.RawData{
					"__id":              llx.StringDataPtr(acl.NetworkAclId),
					"networkAclId":      llx.StringDataPtr(acl.NetworkAclId),
					"networkAclName":    llx.StringDataPtr(acl.NetworkAclName),
					"description":       llx.StringDataPtr(acl.Description),
					"vpcId":             llx.StringDataPtr(acl.VpcId),
					"status":            llx.StringDataPtr(acl.Status),
					"creationTime":      llx.TimeDataPtr(vpcParseTime(acl.CreationTime)),
					"regionId":          llx.StringDataPtr(acl.RegionId),
					"ownerId":           llx.IntDataPtr(acl.OwnerId),
					"ingressAclEntries": llx.ArrayData(ingress, types.Dict),
					"egressAclEntries":  llx.ArrayData(egress, types.Dict),
					"resources":         llx.ArrayData(resources, types.Dict),
					"tags":              llx.MapData(tags, types.String),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, networkAcl)
			}

			total := vpcAclTotalCount(resp.Body.TotalCount)
			if total < 0 || int64(pageNumber)*int64(pageSize) >= total {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

func (r *mqlAlicloudVpcNetworkAcl) id() (string, error) {
	return r.NetworkAclId.Data, nil
}
