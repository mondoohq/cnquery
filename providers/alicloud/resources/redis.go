// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"time"

	rkvclient "github.com/alibabacloud-go/r-kvstore-20150101/v6/client"
	tea "github.com/alibabacloud-go/tea/tea"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
	"go.mondoo.com/mql/v13/types"
)

// redisParseTime converts an RFC3339 timestamp string (as returned by the
// ApsaraDB for Redis API) to a *time.Time, returning nil on a nil or
// unparseable value.
func redisParseTime(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}

// redisTagsToMap flattens the Redis instance tag list into a string map.
func redisTagsToMap(tags *rkvclient.DescribeInstancesResponseBodyInstancesKVStoreInstanceTags) map[string]any {
	res := map[string]any{}
	if tags == nil {
		return res
	}
	for _, t := range tags.Tag {
		if t == nil || t.Key == nil {
			continue
		}
		res[*t.Key] = tea.StringValue(t.Value)
	}
	return res
}

func (r *mqlAlicloudRedis) id() (string, error) {
	return "alicloud.redis", nil
}

// mqlAlicloudRedisInstanceInternal caches the region an instance was
// discovered in so the security-posture accessors can build a region-scoped
// client for the per-instance detail calls, and the native VPC/vSwitch ids so
// the typed accessors can resolve them.
type mqlAlicloudRedisInstanceInternal struct {
	region         string
	cacheRegion    string
	cacheVpcID     string
	cacheVswitchID string
}

func (r *mqlAlicloudRedis) instances() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.RedisClient(region)
		if err != nil {
			return nil, err
		}

		pageNumber := int32(1)
		pageSize := int32(50)
		for {
			resp, err := client.DescribeInstances(&rkvclient.DescribeInstancesRequest{
				RegionId:   tea.String(region),
				PageNumber: tea.Int32(pageNumber),
				PageSize:   tea.Int32(pageSize),
			})
			if err != nil {
				// Skip regions that reject the call (for example a region the
				// account has not activated the service in) rather than failing
				// the whole listing.
				break
			}
			if resp == nil || resp.Body == nil || resp.Body.Instances == nil {
				break
			}

			for _, inst := range resp.Body.Instances.KVStoreInstance {
				if inst == nil || inst.InstanceId == nil {
					continue
				}

				mqlInst, err := CreateResource(r.MqlRuntime, "alicloud.redis.instance", map[string]*llx.RawData{
					"__id":             llx.StringDataPtr(inst.InstanceId),
					"instanceId":       llx.StringDataPtr(inst.InstanceId),
					"instanceName":     llx.StringDataPtr(inst.InstanceName),
					"instanceStatus":   llx.StringDataPtr(inst.InstanceStatus),
					"instanceType":     llx.StringDataPtr(inst.InstanceType),
					"instanceClass":    llx.StringDataPtr(inst.InstanceClass),
					"architectureType": llx.StringDataPtr(inst.ArchitectureType),
					"engineVersion":    llx.StringDataPtr(inst.EngineVersion),
					"regionId":         llx.StringDataPtr(inst.RegionId),
					"zoneId":           llx.StringDataPtr(inst.ZoneId),
					"secondaryZoneId":  llx.StringDataPtr(inst.SecondaryZoneId),
					"networkType":      llx.StringDataPtr(inst.NetworkType),
					"connectionDomain": llx.StringDataPtr(inst.ConnectionDomain),
					"port":             llx.IntDataPtr(inst.Port),
					"privateIp":        llx.StringDataPtr(inst.PrivateIp),
					"capacity":         llx.IntDataPtr(inst.Capacity),
					"bandwidth":        llx.IntDataPtr(inst.Bandwidth),
					"qps":              llx.IntDataPtr(inst.QPS),
					"connections":      llx.IntDataPtr(inst.Connections),
					"chargeType":       llx.StringDataPtr(inst.ChargeType),
					"nodeType":         llx.StringDataPtr(inst.NodeType),
					"packageType":      llx.StringDataPtr(inst.PackageType),
					"editionType":      llx.StringDataPtr(inst.EditionType),
					"resourceGroupId":  llx.StringDataPtr(inst.ResourceGroupId),
					"createTime":       llx.TimeDataPtr(redisParseTime(inst.CreateTime)),
					"endTime":          llx.TimeDataPtr(redisParseTime(inst.EndTime)),
					"tags":             llx.MapData(redisTagsToMap(inst.Tags), types.String),
				})
				if err != nil {
					return nil, err
				}
				resInst := mqlInst.(*mqlAlicloudRedisInstance)
				resInst.region = region
				resInst.cacheRegion = region
				resInst.cacheVpcID = tea.StringValue(inst.VpcId)
				resInst.cacheVswitchID = tea.StringValue(inst.VSwitchId)
				res = append(res, mqlInst)
			}

			total := tea.Int32Value(resp.Body.TotalCount)
			if total == 0 || pageNumber*pageSize >= total {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

func (r *mqlAlicloudRedisInstance) id() (string, error) {
	return r.InstanceId.Data, nil
}

func (r *mqlAlicloudRedisInstance) vpc() (*mqlAlicloudVpcNetwork, error) {
	if r.cacheVpcID == "" {
		r.Vpc.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return resolveVpcNetwork(r.MqlRuntime, r.cacheRegion, r.cacheVpcID)
}

func (r *mqlAlicloudRedisInstance) vswitch() (*mqlAlicloudVpcVswitch, error) {
	if r.cacheVswitchID == "" {
		r.Vswitch.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return resolveVpcVswitch(r.MqlRuntime, r.cacheRegion, r.cacheVswitchID)
}

// redisClient returns a region-scoped Redis client together with this
// instance's ID for the per-instance security-posture detail calls.
func (r *mqlAlicloudRedisInstance) redisClient() (*rkvclient.Client, string, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RedisClient(r.region)
	if err != nil {
		return nil, "", err
	}
	return client, r.InstanceId.Data, nil
}

func (r *mqlAlicloudRedisInstance) sslEnabled() (bool, error) {
	client, id, err := r.redisClient()
	if err != nil {
		return false, err
	}
	resp, err := client.DescribeInstanceSSL(&rkvclient.DescribeInstanceSSLRequest{
		InstanceId: tea.String(id),
	})
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Body == nil || resp.Body.SSLEnabled == nil {
		return false, nil
	}
	return tea.StringValue(resp.Body.SSLEnabled) == "Enable", nil
}

func (r *mqlAlicloudRedisInstance) tdeEnabled() (bool, error) {
	client, id, err := r.redisClient()
	if err != nil {
		return false, err
	}
	resp, err := client.DescribeInstanceTDEStatus(&rkvclient.DescribeInstanceTDEStatusRequest{
		InstanceId: tea.String(id),
	})
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Body == nil || resp.Body.TDEStatus == nil {
		return false, nil
	}
	return tea.StringValue(resp.Body.TDEStatus) == "Enabled", nil
}

func (r *mqlAlicloudRedisInstance) securityIPList() ([]any, error) {
	client, id, err := r.redisClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.DescribeSecurityIps(&rkvclient.DescribeSecurityIpsRequest{
		InstanceId: tea.String(id),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	if resp == nil || resp.Body == nil || resp.Body.SecurityIpGroups == nil {
		return res, nil
	}

	seen := map[string]struct{}{}
	for _, group := range resp.Body.SecurityIpGroups.SecurityIpGroup {
		if group == nil || group.SecurityIpList == nil {
			continue
		}
		for _, ip := range strings.Split(*group.SecurityIpList, ",") {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			if _, ok := seen[ip]; ok {
				continue
			}
			seen[ip] = struct{}{}
			res = append(res, ip)
		}
	}
	return res, nil
}

func (r *mqlAlicloudRedisInstance) securityGroupIds() ([]any, error) {
	client, id, err := r.redisClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.DescribeSecurityGroupConfiguration(&rkvclient.DescribeSecurityGroupConfigurationRequest{
		InstanceId: tea.String(id),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	if resp == nil || resp.Body == nil || resp.Body.Items == nil {
		return res, nil
	}
	for _, rel := range resp.Body.Items.EcsSecurityGroupRelation {
		if rel == nil || rel.SecurityGroupId == nil {
			continue
		}
		res = append(res, *rel.SecurityGroupId)
	}
	return res, nil
}

func (r *mqlAlicloudRedisInstance) authEnabled() (bool, error) {
	client, id, err := r.redisClient()
	if err != nil {
		return false, err
	}
	resp, err := client.DescribeInstanceAttribute(&rkvclient.DescribeInstanceAttributeRequest{
		InstanceId: tea.String(id),
	})
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Body == nil || resp.Body.Instances == nil {
		return false, nil
	}
	for _, attr := range resp.Body.Instances.DBInstanceAttribute {
		if attr == nil || attr.VpcAuthMode == nil {
			continue
		}
		// VpcAuthMode is "Open" when password authentication is enforced and
		// "Close" when password-free access is enabled.
		return tea.StringValue(attr.VpcAuthMode) == "Open", nil
	}
	return false, nil
}
