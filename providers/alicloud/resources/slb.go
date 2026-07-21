// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"fmt"
	"time"

	slbclient "github.com/alibabacloud-go/slb-20140515/v4/client"
	tea "github.com/alibabacloud-go/tea/tea"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
	"go.mondoo.com/mql/v13/types"
)

// slbParseTime parses an RFC3339 timestamp returned by the SLB API, returning
// nil when the input is nil or cannot be parsed.
func slbParseTime(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}

// slbLoadBalancerTags flattens the SLB describe-load-balancers tag list into a
// string map.
func slbLoadBalancerTags(tags *slbclient.DescribeLoadBalancersResponseBodyLoadBalancersLoadBalancerTags) map[string]any {
	res := map[string]any{}
	if tags == nil {
		return res
	}
	for _, t := range tags.Tag {
		if t == nil || t.TagKey == nil {
			continue
		}
		res[*t.TagKey] = tea.StringValue(t.TagValue)
	}
	return res
}

// slbListenerConfig marshals the protocol-specific listener config that is set
// on the listener into a generic dict. Exactly one of the four config structs
// is populated per listener, keyed by the listener protocol.
func slbListenerConfig(l *slbclient.DescribeLoadBalancerListenersResponseBodyListeners) any {
	var cfg any
	switch {
	case l.HTTPListenerConfig != nil:
		cfg = l.HTTPListenerConfig
	case l.HTTPSListenerConfig != nil:
		cfg = l.HTTPSListenerConfig
	case l.TCPListenerConfig != nil:
		cfg = l.TCPListenerConfig
	case l.UDPListenerConfig != nil:
		cfg = l.UDPListenerConfig
	default:
		return nil
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

func (r *mqlAlicloudSlb) id() (string, error) {
	return "alicloud.slb", nil
}

type mqlAlicloudSlbLoadBalancerInternal struct {
	region         string
	cacheRegion    string
	cacheVpcID     string
	cacheVswitchID string
}

func (r *mqlAlicloudSlb) loadBalancers() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.SlbClient(region)
		if err != nil {
			return nil, err
		}

		pageNumber := int32(1)
		pageSize := int32(100)
		firstPage := true
		for {
			resp, err := client.DescribeLoadBalancers(&slbclient.DescribeLoadBalancersRequest{
				RegionId:   tea.String(region),
				PageNumber: tea.Int32(pageNumber),
				PageSize:   tea.Int32(pageSize),
			})
			if err != nil {
				// A first-page error means the region has no SLB or the
				// credential lacks access there — skip it. A later-page error
				// means the region is reachable and the failure is real; surface
				// it rather than silently truncating the list (matches ALB/NLB).
				if firstPage {
					break
				}
				return nil, err
			}
			firstPage = false
			if resp == nil || resp.Body == nil || resp.Body.LoadBalancers == nil {
				break
			}

			lbs := resp.Body.LoadBalancers.LoadBalancer
			for _, lb := range lbs {
				if lb == nil || lb.LoadBalancerId == nil {
					continue
				}

				resource, err := CreateResource(r.MqlRuntime, "alicloud.slb.loadBalancer", map[string]*llx.RawData{
					"__id":                         llx.StringDataPtr(lb.LoadBalancerId),
					"loadBalancerId":               llx.StringDataPtr(lb.LoadBalancerId),
					"loadBalancerName":             llx.StringDataPtr(lb.LoadBalancerName),
					"address":                      llx.StringDataPtr(lb.Address),
					"addressType":                  llx.StringDataPtr(lb.AddressType),
					"addressIPVersion":             llx.StringDataPtr(lb.AddressIPVersion),
					"status":                       llx.StringDataPtr(lb.LoadBalancerStatus),
					"regionId":                     llx.StringDataPtr(lb.RegionId),
					"masterZoneId":                 llx.StringDataPtr(lb.MasterZoneId),
					"slaveZoneId":                  llx.StringDataPtr(lb.SlaveZoneId),
					"networkType":                  llx.StringDataPtr(lb.NetworkType),
					"internetChargeType":           llx.StringDataPtr(lb.InternetChargeType),
					"instanceChargeType":           llx.StringDataPtr(lb.InstanceChargeType),
					"loadBalancerSpec":             llx.StringDataPtr(lb.LoadBalancerSpec),
					"payType":                      llx.StringDataPtr(lb.PayType),
					"bandwidth":                    llx.IntDataPtr(lb.Bandwidth),
					"createTime":                   llx.TimeDataPtr(slbParseTime(lb.CreateTime)),
					"deleteProtection":             llx.StringDataPtr(lb.DeleteProtection),
					"modificationProtectionStatus": llx.StringDataPtr(lb.ModificationProtectionStatus),
					"modificationProtectionReason": llx.StringDataPtr(lb.ModificationProtectionReason),
					"resourceGroupId":              llx.StringDataPtr(lb.ResourceGroupId),
					"tags":                         llx.MapData(slbLoadBalancerTags(lb.Tags), types.String),
				})
				if err != nil {
					return nil, err
				}

				mqlLb := resource.(*mqlAlicloudSlbLoadBalancer)
				mqlLb.region = region
				mqlLb.cacheRegion = region
				mqlLb.cacheVpcID = tea.StringValue(lb.VpcId)
				mqlLb.cacheVswitchID = tea.StringValue(lb.VSwitchId)
				res = append(res, mqlLb)
			}

			if len(lbs) == 0 || int(pageNumber)*int(pageSize) >= int(tea.Int32Value(resp.Body.TotalCount)) {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

func (r *mqlAlicloudSlbLoadBalancer) id() (string, error) {
	return r.LoadBalancerId.Data, nil
}

func (r *mqlAlicloudSlbLoadBalancer) vpc() (*mqlAlicloudVpcNetwork, error) {
	if r.cacheVpcID == "" {
		r.Vpc.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return resolveVpcNetwork(r.MqlRuntime, r.cacheRegion, r.cacheVpcID)
}

func (r *mqlAlicloudSlbLoadBalancer) vswitch() (*mqlAlicloudVpcVswitch, error) {
	if r.cacheVswitchID == "" {
		r.Vswitch.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return resolveVpcVswitch(r.MqlRuntime, r.cacheRegion, r.cacheVswitchID)
}

func (r *mqlAlicloudSlbLoadBalancer) listeners() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	lbId := r.LoadBalancerId.Data
	client, err := conn.SlbClient(r.region)
	if err != nil {
		return nil, err
	}

	res := []any{}
	var nextToken *string
	for {
		resp, err := client.DescribeLoadBalancerListeners(&slbclient.DescribeLoadBalancerListenersRequest{
			RegionId:       tea.String(r.region),
			LoadBalancerId: []*string{tea.String(lbId)},
			MaxResults:     tea.Int32(100),
			NextToken:      nextToken,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil {
			break
		}

		for _, l := range resp.Body.Listeners {
			if l == nil {
				continue
			}

			protocol := tea.StringValue(l.ListenerProtocol)
			port := tea.Int32Value(l.ListenerPort)

			args := map[string]*llx.RawData{
				"__id":                llx.StringData(fmt.Sprintf("%s/%s/%d", lbId, protocol, port)),
				"loadBalancerId":      llx.StringData(lbId),
				"protocol":            llx.StringDataPtr(l.ListenerProtocol),
				"listenerPort":        llx.IntDataPtr(l.ListenerPort),
				"backendServerPort":   llx.IntDataPtr(l.BackendServerPort),
				"status":              llx.StringDataPtr(l.Status),
				"bandwidth":           llx.IntDataPtr(l.Bandwidth),
				"scheduler":           llx.StringDataPtr(l.Scheduler),
				"description":         llx.StringDataPtr(l.Description),
				"aclStatus":           llx.StringDataPtr(l.AclStatus),
				"aclType":             llx.StringDataPtr(l.AclType),
				"aclId":               llx.StringDataPtr(l.AclId),
				"aclIds":              llx.ArrayData(llx.TArr2Raw(convertSlbStrPtrs(l.AclIds)), types.String),
				"vServerGroupId":      llx.StringDataPtr(l.VServerGroupId),
				"config":              llx.DictData(slbListenerConfig(l)),
				"tlsCipherPolicy":     llx.StringDataPtr(nil),
				"serverCertificateId": llx.StringDataPtr(nil),
				"caCertificateId":     llx.StringDataPtr(nil),
				"enableHttp2":         llx.StringDataPtr(nil),
			}

			if https := l.HTTPSListenerConfig; https != nil {
				args["tlsCipherPolicy"] = llx.StringDataPtr(https.TLSCipherPolicy)
				args["serverCertificateId"] = llx.StringDataPtr(https.ServerCertificateId)
				args["caCertificateId"] = llx.StringDataPtr(https.CACertificateId)
				args["enableHttp2"] = llx.StringDataPtr(https.EnableHttp2)
			}

			listener, err := CreateResource(r.MqlRuntime, "alicloud.slb.listener", args)
			if err != nil {
				return nil, err
			}
			res = append(res, listener)
		}

		if resp.Body.NextToken == nil || *resp.Body.NextToken == "" {
			break
		}
		nextToken = resp.Body.NextToken
	}
	return res, nil
}

func (r *mqlAlicloudSlbLoadBalancer) backendServers() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.SlbClient(r.region)
	if err != nil {
		return nil, err
	}

	resp, err := client.DescribeLoadBalancerAttribute(&slbclient.DescribeLoadBalancerAttributeRequest{
		RegionId:       tea.String(r.region),
		LoadBalancerId: tea.String(r.LoadBalancerId.Data),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	if resp == nil || resp.Body == nil || resp.Body.BackendServers == nil {
		return res, nil
	}

	for _, s := range resp.Body.BackendServers.BackendServer {
		if s == nil {
			continue
		}
		res = append(res, map[string]any{
			"serverId":    tea.StringValue(s.ServerId),
			"serverIp":    tea.StringValue(s.ServerIp),
			"weight":      int64(tea.Int32Value(s.Weight)),
			"type":        tea.StringValue(s.Type),
			"description": tea.StringValue(s.Description),
		})
	}
	return res, nil
}

func (r *mqlAlicloudSlbListener) id() (string, error) {
	return fmt.Sprintf("%s/%s/%d", r.LoadBalancerId.Data, r.Protocol.Data, r.ListenerPort.Data), nil
}

// convertSlbStrPtrs dereferences a slice of string pointers, skipping nil
// elements.
func convertSlbStrPtrs(in []*string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == nil {
			continue
		}
		out = append(out, *s)
	}
	return out
}
