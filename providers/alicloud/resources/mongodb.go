// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ddsclient "github.com/alibabacloud-go/dds-20151201/v9/client"
	tea "github.com/alibabacloud-go/tea/tea"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
	"go.mondoo.com/mql/v13/types"
)

// mongodbParseTime parses an Alibaba Cloud timestamp string into a *time.Time.
// ApsaraDB for MongoDB returns times in a few ISO 8601 variants (with or
// without seconds), so several layouts are attempted. Returns nil on a nil or
// unparseable input.
func mongodbParseTime(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04Z",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, *s); err == nil {
			return &t
		}
	}
	return nil
}

// mongodbTagsToMap converts the instance tag list into a name/value map.
func mongodbTagsToMap(tags *ddsclient.DescribeDBInstancesResponseBodyDBInstancesDBInstanceTags) map[string]any {
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

func (r *mqlAlicloudMongodb) id() (string, error) {
	return "alicloud.mongodb", nil
}

func (r *mqlAlicloudMongodb) instances() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.MongoDBClient(region)
		if err != nil {
			return nil, err
		}

		pageNumber := int32(1)
		pageSize := int32(100)
		for {
			resp, err := client.DescribeDBInstances(&ddsclient.DescribeDBInstancesRequest{
				RegionId:   tea.String(region),
				PageNumber: tea.Int32(pageNumber),
				PageSize:   tea.Int32(pageSize),
			})
			if err != nil {
				// a region may be un-activated or access-denied; skip it rather than failing the whole scan
				break
			}
			if resp == nil || resp.Body == nil || resp.Body.DBInstances == nil {
				break
			}

			items := resp.Body.DBInstances.DBInstance
			for _, inst := range items {
				if inst == nil || inst.DBInstanceId == nil {
					continue
				}
				id := tea.StringValue(inst.DBInstanceId)

				mqlInst, err := CreateResource(r.MqlRuntime, "alicloud.mongodb.instance", map[string]*llx.RawData{
					"__id":                  llx.StringData(id),
					"dbInstanceId":          llx.StringDataPtr(inst.DBInstanceId),
					"dbInstanceDescription": llx.StringDataPtr(inst.DBInstanceDescription),
					"dbInstanceType":        llx.StringDataPtr(inst.DBInstanceType),
					"dbInstanceClass":       llx.StringDataPtr(inst.DBInstanceClass),
					"dbInstanceStorage":     llx.IntData(tea.Int32Value(inst.DBInstanceStorage)),
					"engine":                llx.StringDataPtr(inst.Engine),
					"engineVersion":         llx.StringDataPtr(inst.EngineVersion),
					"dbInstanceStatus":      llx.StringDataPtr(inst.DBInstanceStatus),
					"regionId":              llx.StringDataPtr(inst.RegionId),
					"zoneId":                llx.StringDataPtr(inst.ZoneId),
					"secondaryZoneId":       llx.StringDataPtr(inst.SecondaryZoneId),
					"hiddenZoneId":          llx.StringDataPtr(inst.HiddenZoneId),
					"networkType":           llx.StringDataPtr(inst.NetworkType),
					"chargeType":            llx.StringDataPtr(inst.ChargeType),
					"storageType":           llx.StringDataPtr(inst.StorageType),
					"replicationFactor":     llx.StringDataPtr(inst.ReplicationFactor),
					"vpcAuthMode":           llx.StringDataPtr(inst.VpcAuthMode),
					"backupRetentionPolicy": llx.IntData(tea.Int32Value(inst.BackupRetentionPolicy)),
					"capacityUnit":          llx.StringDataPtr(inst.CapacityUnit),
					"kindCode":              llx.StringDataPtr(inst.KindCode),
					"createTime":            llx.TimeDataPtr(mongodbParseTime(inst.CreationTime)),
					"expireTime":            llx.TimeDataPtr(mongodbParseTime(inst.ExpireTime)),
					"destroyTime":           llx.TimeDataPtr(mongodbParseTime(inst.DestroyTime)),
					"releaseTime":           llx.TimeDataPtr(mongodbParseTime(inst.ReleaseTime)),
					"lastDowngradeTime":     llx.StringDataPtr(inst.LastDowngradeTime),
					"lockMode":              llx.StringDataPtr(inst.LockMode),
					"resourceGroupId":       llx.StringDataPtr(inst.ResourceGroupId),
					"tags":                  llx.MapData(mongodbTagsToMap(inst.Tags), types.String),
				})
				if err != nil {
					return nil, err
				}
				m := mqlInst.(*mqlAlicloudMongodbInstance)
				m.region = region
				m.cacheRegion = region
				m.instanceId = id
				res = append(res, mqlInst)
			}

			// Stop when the current page did not fill or all instances were seen.
			total := tea.Int32Value(resp.Body.TotalCount)
			if len(items) == 0 || int64(pageNumber)*int64(pageSize) >= int64(total) {
				break
			}
			pageNumber++
		}
	}

	return res, nil
}

// mqlAlicloudMongodbInstanceInternal caches the per-instance detail responses so
// the several detail-derived accessors each trigger at most one API call.
type mqlAlicloudMongodbInstanceInternal struct {
	region     string
	instanceId string

	cacheRegion    string
	cacheVpcID     string
	cacheVswitchID string

	attrLock    sync.Mutex
	attrFetched atomic.Bool
	attr        *ddsclient.DescribeDBInstanceAttributeResponseBodyDBInstancesDBInstance

	sslOnce sync.Once
	sslBody *ddsclient.DescribeDBInstanceSSLResponseBody
	sslErr  error
}

func (r *mqlAlicloudMongodbInstance) id() (string, error) {
	return r.DbInstanceId.Data, nil
}

// attribute lazily fetches and caches the DescribeDBInstanceAttribute detail. It
// backs several attribute-derived accessors, so it batches into a single API
// call. A transient error is not cached: attrFetched is only set on success, so
// a later access retries the call.
func (r *mqlAlicloudMongodbInstance) attribute() (*ddsclient.DescribeDBInstanceAttributeResponseBodyDBInstancesDBInstance, error) {
	if r.attrFetched.Load() {
		return r.attr, nil
	}
	r.attrLock.Lock()
	defer r.attrLock.Unlock()
	if r.attrFetched.Load() {
		return r.attr, nil
	}

	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.MongoDBClient(r.region)
	if err != nil {
		return nil, err
	}
	resp, err := client.DescribeDBInstanceAttribute(&ddsclient.DescribeDBInstanceAttributeRequest{
		DBInstanceId: tea.String(r.instanceId),
	})
	if err != nil {
		return nil, err
	}
	if resp != nil && resp.Body != nil && resp.Body.DBInstances != nil && len(resp.Body.DBInstances.DBInstance) > 0 {
		r.attr = resp.Body.DBInstances.DBInstance[0]
		r.cacheVpcID = tea.StringValue(r.attr.VPCId)
		r.cacheVswitchID = tea.StringValue(r.attr.VSwitchId)
	}
	r.attrFetched.Store(true)
	return r.attr, nil
}

func (r *mqlAlicloudMongodbInstance) vpc() (*mqlAlicloudVpcNetwork, error) {
	if _, err := r.attribute(); err != nil {
		return nil, err
	}
	if r.cacheVpcID == "" {
		r.Vpc.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return resolveVpcNetwork(r.MqlRuntime, r.cacheRegion, r.cacheVpcID)
}

func (r *mqlAlicloudMongodbInstance) vswitch() (*mqlAlicloudVpcVswitch, error) {
	if _, err := r.attribute(); err != nil {
		return nil, err
	}
	if r.cacheVswitchID == "" {
		r.Vswitch.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return resolveVpcVswitch(r.MqlRuntime, r.cacheRegion, r.cacheVswitchID)
}

func (r *mqlAlicloudMongodbInstance) storageEngine() (string, error) {
	attr, err := r.attribute()
	if err != nil || attr == nil {
		return "", err
	}
	return tea.StringValue(attr.StorageEngine), nil
}

func (r *mqlAlicloudMongodbInstance) protocolType() (string, error) {
	attr, err := r.attribute()
	if err != nil || attr == nil {
		return "", err
	}
	return tea.StringValue(attr.ProtocolType), nil
}

func (r *mqlAlicloudMongodbInstance) readonlyReplicas() (string, error) {
	attr, err := r.attribute()
	if err != nil || attr == nil {
		return "", err
	}
	return tea.StringValue(attr.ReadonlyReplicas), nil
}

func (r *mqlAlicloudMongodbInstance) maintainStartTime() (string, error) {
	attr, err := r.attribute()
	if err != nil || attr == nil {
		return "", err
	}
	return tea.StringValue(attr.MaintainStartTime), nil
}

func (r *mqlAlicloudMongodbInstance) maintainEndTime() (string, error) {
	attr, err := r.attribute()
	if err != nil || attr == nil {
		return "", err
	}
	return tea.StringValue(attr.MaintainEndTime), nil
}

func (r *mqlAlicloudMongodbInstance) encrypted() (bool, error) {
	attr, err := r.attribute()
	if err != nil || attr == nil {
		return false, err
	}
	return tea.BoolValue(attr.Encrypted), nil
}

func (r *mqlAlicloudMongodbInstance) encryptionKey() (string, error) {
	attr, err := r.attribute()
	if err != nil || attr == nil {
		return "", err
	}
	return tea.StringValue(attr.EncryptionKey), nil
}

func (r *mqlAlicloudMongodbInstance) releaseProtection() (bool, error) {
	attr, err := r.attribute()
	if err != nil || attr == nil {
		return false, err
	}
	return tea.BoolValue(attr.DBInstanceReleaseProtection), nil
}

func (r *mqlAlicloudMongodbInstance) currentKernelVersion() (string, error) {
	attr, err := r.attribute()
	if err != nil || attr == nil {
		return "", err
	}
	return tea.StringValue(attr.CurrentKernelVersion), nil
}

// sslInfo lazily fetches and caches the DescribeDBInstanceSSL response, shared by
// sslEnabled and sslExpireTime.
func (r *mqlAlicloudMongodbInstance) sslInfo() (*ddsclient.DescribeDBInstanceSSLResponseBody, error) {
	r.sslOnce.Do(func() {
		conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
		client, err := conn.MongoDBClient(r.region)
		if err != nil {
			r.sslErr = err
			return
		}
		resp, err := client.DescribeDBInstanceSSL(&ddsclient.DescribeDBInstanceSSLRequest{
			DBInstanceId: tea.String(r.instanceId),
		})
		if err != nil {
			r.sslErr = err
			return
		}
		if resp != nil {
			r.sslBody = resp.Body
		}
	})
	return r.sslBody, r.sslErr
}

func (r *mqlAlicloudMongodbInstance) sslEnabled() (bool, error) {
	body, err := r.sslInfo()
	if err != nil || body == nil {
		return false, err
	}
	return tea.StringValue(body.SSLStatus) == "Open", nil
}

func (r *mqlAlicloudMongodbInstance) sslExpireTime() (*time.Time, error) {
	body, err := r.sslInfo()
	if err != nil || body == nil {
		return nil, err
	}
	return mongodbParseTime(body.SSLExpiredTime), nil
}

func (r *mqlAlicloudMongodbInstance) tdeEnabled() (bool, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.MongoDBClient(r.region)
	if err != nil {
		return false, err
	}
	resp, err := client.DescribeDBInstanceTDEInfo(&ddsclient.DescribeDBInstanceTDEInfoRequest{
		DBInstanceId: tea.String(r.instanceId),
	})
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Body == nil {
		return false, nil
	}
	return tea.StringValue(resp.Body.TDEStatus) == "enabled", nil
}

func (r *mqlAlicloudMongodbInstance) securityIPList() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.MongoDBClient(r.region)
	if err != nil {
		return nil, err
	}
	resp, err := client.DescribeSecurityIps(&ddsclient.DescribeSecurityIpsRequest{
		DBInstanceId: tea.String(r.instanceId),
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Body == nil {
		return []any{}, nil
	}

	seen := map[string]struct{}{}
	res := []any{}
	appendIps := func(raw *string) {
		if raw == nil {
			return
		}
		for _, ip := range strings.Split(*raw, ",") {
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

	appendIps(resp.Body.SecurityIps)
	if resp.Body.SecurityIpGroups != nil {
		for _, g := range resp.Body.SecurityIpGroups.SecurityIpGroup {
			if g == nil {
				continue
			}
			appendIps(g.SecurityIpList)
		}
	}
	return res, nil
}

func (r *mqlAlicloudMongodbInstance) securityGroupIds() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.MongoDBClient(r.region)
	if err != nil {
		return nil, err
	}
	resp, err := client.DescribeSecurityGroupConfiguration(&ddsclient.DescribeSecurityGroupConfigurationRequest{
		DBInstanceId: tea.String(r.instanceId),
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Body == nil || resp.Body.Items == nil {
		return []any{}, nil
	}

	res := []any{}
	for _, rel := range resp.Body.Items.RdsEcsSecurityGroupRel {
		if rel == nil || rel.SecurityGroupId == nil {
			continue
		}
		res = append(res, tea.StringValue(rel.SecurityGroupId))
	}
	return res, nil
}

func (r *mqlAlicloudMongodbInstance) auditPolicyEnabled() (bool, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.MongoDBClient(r.region)
	if err != nil {
		return false, err
	}
	resp, err := client.DescribeAuditPolicy(&ddsclient.DescribeAuditPolicyRequest{
		DBInstanceId: tea.String(r.instanceId),
	})
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Body == nil {
		return false, nil
	}
	return tea.StringValue(resp.Body.LogAuditStatus) == "Enable", nil
}
