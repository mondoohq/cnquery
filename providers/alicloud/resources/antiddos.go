// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	ddoscooclient "github.com/alibabacloud-go/ddoscoo-20200101/v5/client"
	tea "github.com/alibabacloud-go/tea/tea"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlAlicloudAntiddos) id() (string, error) {
	return "alicloud.antiddos", nil
}

func (r *mqlAlicloudAntiddos) instances() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)

	res := []any{}
	// Anti-DDoS is a center service; try both centers and skip a failing one,
	// but surface the error only if NEITHER center responded, so a transient
	// outage is not silently reported as "no Anti-DDoS".
	var lastErr error
	succeeded := 0
	for _, region := range alicloudCenterRegions {
		client, err := conn.DdoscooClient(region)
		if err != nil {
			lastErr = err
			continue
		}

		pageNumber := 1
		pageSize := 50
		centerOk := false
		for {
			resp, err := client.DescribeInstances(&ddoscooclient.DescribeInstancesRequest{
				PageNumber: tea.String(strconv.Itoa(pageNumber)),
				PageSize:   tea.String(strconv.Itoa(pageSize)),
			})
			if err != nil {
				if !centerOk {
					// first page failed: wrong partition or transient. Record it
					// and move on; total failure is surfaced after the loop.
					lastErr = err
				} else {
					// a mid-pagination failure in a reachable center is a real
					// error, not a missing-service case
					return nil, err
				}
				break
			}
			centerOk = true
			if resp == nil || resp.Body == nil {
				break
			}
			items := resp.Body.Instances
			for _, inst := range items {
				if inst == nil || inst.InstanceId == nil {
					continue
				}
				mqlInstance, err := newAntiddosInstance(r.MqlRuntime, region, inst)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}
			if len(items) < pageSize {
				break
			}
			pageNumber++
		}
		if centerOk {
			succeeded++
		}
	}
	if succeeded == 0 && lastErr != nil {
		return nil, lastErr
	}
	return res, nil
}

// mqlAlicloudAntiddosInstanceInternal caches the center region and instance id
// used by the child accessors.
type mqlAlicloudAntiddosInstanceInternal struct {
	region     string
	instanceId string
}

func newAntiddosInstance(runtime *plugin.Runtime, region string, inst *ddoscooclient.DescribeInstancesResponseBodyInstances) (*mqlAlicloudAntiddosInstance, error) {
	instanceID := tea.StringValue(inst.InstanceId)
	resource, err := CreateResource(runtime, "alicloud.antiddos.instance", map[string]*llx.RawData{
		"__id":       llx.StringData(region + "/" + instanceID),
		"regionId":   llx.StringData(region),
		"instanceId": llx.StringData(instanceID),
		"remark":     llx.StringDataPtr(inst.Remark),
		"status":     llx.IntData(int64(tea.Int32Value(inst.Status))),
		"edition":    llx.IntData(int64(tea.Int32Value(inst.Edition))),
		"enabled":    llx.BoolData(tea.Int32Value(inst.Enabled) == 1),
		"ipMode":     llx.StringDataPtr(inst.IpMode),
		"ipVersion":  llx.StringDataPtr(inst.IpVersion),
		"ip":         llx.StringDataPtr(inst.Ip),
		"debtStatus": llx.IntData(int64(tea.Int32Value(inst.DebtStatus))),
		"expireTime": llx.TimeDataPtr(configEpochMillis(inst.ExpireTime)),
		"createTime": llx.TimeDataPtr(configEpochMillis(inst.CreateTime)),
	})
	if err != nil {
		return nil, err
	}
	mqlInstance := resource.(*mqlAlicloudAntiddosInstance)
	mqlInstance.region = region
	mqlInstance.instanceId = instanceID
	return mqlInstance, nil
}

func (r *mqlAlicloudAntiddosInstance) id() (string, error) {
	return r.region + "/" + r.instanceId, nil
}

func (r *mqlAlicloudAntiddosInstance) webRules() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.DdoscooClient(r.region)
	if err != nil {
		return nil, err
	}

	res := []any{}
	pageNumber := int32(1)
	pageSize := int32(10)
	for {
		resp, err := client.DescribeWebRules(&ddoscooclient.DescribeWebRulesRequest{
			InstanceIds: []*string{tea.String(r.instanceId)},
			PageNumber:  tea.Int32(pageNumber),
			PageSize:    tea.Int32(pageSize),
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil {
			break
		}
		items := resp.Body.WebRules
		for _, wr := range items {
			if wr == nil || wr.Domain == nil {
				continue
			}
			mqlRule, err := newAntiddosWebRule(r.MqlRuntime, r.region, r.instanceId, wr)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRule)
		}
		if len(items) < int(pageSize) {
			break
		}
		pageNumber++
	}
	return res, nil
}

func newAntiddosWebRule(runtime *plugin.Runtime, region, instanceID string, wr *ddoscooclient.DescribeWebRulesResponseBodyWebRules) (*mqlAlicloudAntiddosWebRule, error) {
	proxyTypes := []any{}
	for _, pt := range wr.ProxyTypes {
		if pt == nil {
			continue
		}
		proxyTypes = append(proxyTypes, map[string]any{
			"proxyType":  tea.StringValue(pt.ProxyType),
			"proxyPorts": strPtrsToAny(pt.ProxyPorts),
		})
	}
	realServers := []any{}
	for _, rs := range wr.RealServers {
		if rs == nil {
			continue
		}
		realServers = append(realServers, map[string]any{
			"realServer": tea.StringValue(rs.RealServer),
			"rsType":     int64(tea.Int32Value(rs.RsType)),
		})
	}

	resource, err := CreateResource(runtime, "alicloud.antiddos.webRule", map[string]*llx.RawData{
		"__id":           llx.StringData(region + "/" + instanceID + "/" + tea.StringValue(wr.Domain)),
		"regionId":       llx.StringData(region),
		"instanceId":     llx.StringData(instanceID),
		"domain":         llx.StringDataPtr(wr.Domain),
		"cname":          llx.StringDataPtr(wr.Cname),
		"ccEnabled":      llx.BoolDataPtr(wr.CcEnabled),
		"ccRuleEnabled":  llx.BoolDataPtr(wr.CcRuleEnabled),
		"certName":       llx.StringDataPtr(wr.CertName),
		"certExpireTime": llx.TimeDataPtr(configEpochMillis(wr.CertExpireTime)),
		"penalized":      llx.BoolDataPtr(wr.PunishStatus),
		"proxyTypes":     llx.ArrayData(proxyTypes, types.Dict),
		"realServers":    llx.ArrayData(realServers, types.Dict),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAlicloudAntiddosWebRule), nil
}

func (r *mqlAlicloudAntiddosWebRule) id() (string, error) {
	return r.RegionId.Data + "/" + r.InstanceId.Data + "/" + r.Domain.Data, nil
}

func (r *mqlAlicloudAntiddosInstance) networkRules() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.DdoscooClient(r.region)
	if err != nil {
		return nil, err
	}

	res := []any{}
	pageNumber := int32(1)
	pageSize := int32(10)
	for {
		resp, err := client.DescribeNetworkRules(&ddoscooclient.DescribeNetworkRulesRequest{
			InstanceId: tea.String(r.instanceId),
			PageNumber: tea.Int32(pageNumber),
			PageSize:   tea.Int32(pageSize),
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil {
			break
		}
		items := resp.Body.NetworkRules
		for _, nr := range items {
			if nr == nil || nr.FrontendPort == nil {
				continue
			}
			resource, err := CreateResource(r.MqlRuntime, "alicloud.antiddos.networkRule", map[string]*llx.RawData{
				"__id": llx.StringData(r.region + "/" + r.instanceId + "/" +
					tea.StringValue(nr.Protocol) + "/" + strconv.Itoa(int(tea.Int32Value(nr.FrontendPort)))),
				"regionId":     llx.StringData(r.region),
				"instanceId":   llx.StringData(r.instanceId),
				"protocol":     llx.StringDataPtr(nr.Protocol),
				"frontendPort": llx.IntData(int64(tea.Int32Value(nr.FrontendPort))),
				"backendPort":  llx.IntData(int64(tea.Int32Value(nr.BackendPort))),
				"realServers":  llx.ArrayData(strPtrsToAny(nr.RealServers), types.String),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, resource)
		}
		if len(items) < int(pageSize) {
			break
		}
		pageNumber++
	}
	return res, nil
}

func (r *mqlAlicloudAntiddosNetworkRule) id() (string, error) {
	return r.RegionId.Data + "/" + r.InstanceId.Data + "/" + r.Protocol.Data + "/" + strconv.FormatInt(r.FrontendPort.Data, 10), nil
}
