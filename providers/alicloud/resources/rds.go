// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"
	"sync"
	"time"

	rdsclient "github.com/alibabacloud-go/rds-20140815/v11/client"
	tea "github.com/alibabacloud-go/tea/tea"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
)

// rdsParseTime parses an RFC3339 timestamp returned by the RDS API, returning
// nil when the input is nil or cannot be parsed.
func rdsParseTime(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}

func (r *mqlAlicloudRds) id() (string, error) {
	return "alicloud.rds", nil
}

type mqlAlicloudRdsInstanceInternal struct {
	region     string
	instanceId string

	cacheRegion    string
	cacheVpcID     string
	cacheVswitchID string

	attrLock sync.Mutex
	attrDone bool
	attr     *rdsclient.DescribeDBInstanceAttributeResponseBodyItemsDBInstanceAttribute

	sslLock sync.Mutex
	sslDone bool
	ssl     *rdsclient.DescribeDBInstanceSSLResponseBody
}

func (r *mqlAlicloudRds) instances() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.RdsClient(region)
		if err != nil {
			return nil, err
		}

		pageNumber := int32(1)
		pageSize := int32(100)
		for {
			resp, err := client.DescribeDBInstances(&rdsclient.DescribeDBInstancesRequest{
				RegionId:   tea.String(region),
				PageNumber: tea.Int32(pageNumber),
				PageSize:   tea.Int32(pageSize),
			})
			if err != nil {
				// A region may not have RDS enabled or the credential may lack
				// access there. Skip it rather than failing the whole list.
				break
			}
			if resp == nil || resp.Body == nil || resp.Body.Items == nil {
				break
			}

			items := resp.Body.Items.DBInstance
			for _, inst := range items {
				if inst == nil || inst.DBInstanceId == nil {
					continue
				}

				id := tea.StringValue(inst.DBInstanceId) + "/" + region
				resource, err := CreateResource(r.MqlRuntime, "alicloud.rds.instance", map[string]*llx.RawData{
					"__id":                  llx.StringData(id),
					"dbInstanceId":          llx.StringDataPtr(inst.DBInstanceId),
					"dbInstanceDescription": llx.StringDataPtr(inst.DBInstanceDescription),
					"engine":                llx.StringDataPtr(inst.Engine),
					"engineVersion":         llx.StringDataPtr(inst.EngineVersion),
					"dbInstanceStatus":      llx.StringDataPtr(inst.DBInstanceStatus),
					"dbInstanceType":        llx.StringDataPtr(inst.DBInstanceType),
					"dbInstanceClass":       llx.StringDataPtr(inst.DBInstanceClass),
					"dbInstanceStorageType": llx.StringDataPtr(inst.DBInstanceStorageType),
					"dbInstanceNetType":     llx.StringDataPtr(inst.DBInstanceNetType),
					"connectionMode":        llx.StringDataPtr(inst.ConnectionMode),
					"connectionString":      llx.StringDataPtr(inst.ConnectionString),
					"regionId":              llx.StringDataPtr(inst.RegionId),
					"zoneId":                llx.StringDataPtr(inst.ZoneId),
					"instanceNetworkType":   llx.StringDataPtr(inst.InstanceNetworkType),
					"payType":               llx.StringDataPtr(inst.PayType),
					"createTime":            llx.TimeDataPtr(rdsParseTime(inst.CreateTime)),
					"expireTime":            llx.TimeDataPtr(rdsParseTime(inst.ExpireTime)),
					"lockMode":              llx.StringDataPtr(inst.LockMode),
					"lockReason":            llx.StringDataPtr(inst.LockReason),
					"category":              llx.StringDataPtr(inst.Category),
					"deletionProtection":    llx.BoolDataPtr(inst.DeletionProtection),
					"masterInstanceId":      llx.StringDataPtr(inst.MasterInstanceId),
					"resourceGroupId":       llx.StringDataPtr(inst.ResourceGroupId),
				})
				if err != nil {
					return nil, err
				}

				mqlInst := resource.(*mqlAlicloudRdsInstance)
				mqlInst.region = region
				mqlInst.instanceId = tea.StringValue(inst.DBInstanceId)
				mqlInst.cacheRegion = region
				mqlInst.cacheVpcID = tea.StringValue(inst.VpcId)
				mqlInst.cacheVswitchID = tea.StringValue(inst.VSwitchId)
				res = append(res, mqlInst)
			}

			if len(items) == 0 || int(pageNumber)*int(pageSize) >= int(tea.Int32Value(resp.Body.TotalRecordCount)) {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

func (r *mqlAlicloudRdsInstance) id() (string, error) {
	return r.instanceId + "/" + r.region, nil
}

// fetchAttribute lazily loads the per-instance DescribeDBInstanceAttribute
// detail and caches it. Returns nil when the call fails or returns no detail,
// so callers fall back to their safe default.
func (r *mqlAlicloudRdsInstance) fetchAttribute() *rdsclient.DescribeDBInstanceAttributeResponseBodyItemsDBInstanceAttribute {
	r.attrLock.Lock()
	defer r.attrLock.Unlock()
	if r.attrDone {
		return r.attr
	}
	r.attrDone = true

	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RdsClient(r.region)
	if err != nil {
		return nil
	}
	resp, err := client.DescribeDBInstanceAttribute(&rdsclient.DescribeDBInstanceAttributeRequest{
		DBInstanceId: tea.String(r.instanceId),
	})
	if err != nil || resp == nil || resp.Body == nil || resp.Body.Items == nil {
		return nil
	}
	attrs := resp.Body.Items.DBInstanceAttribute
	if len(attrs) == 0 {
		return nil
	}
	r.attr = attrs[0]
	return r.attr
}

func (r *mqlAlicloudRdsInstance) dbInstanceStorage() (int64, error) {
	attr := r.fetchAttribute()
	if attr == nil {
		return 0, nil
	}
	return int64(tea.Int32Value(attr.DBInstanceStorage)), nil
}

func (r *mqlAlicloudRdsInstance) port() (int64, error) {
	attr := r.fetchAttribute()
	if attr == nil || attr.Port == nil {
		return 0, nil
	}
	p, err := strconv.Atoi(strings.TrimSpace(*attr.Port))
	if err != nil {
		return 0, nil
	}
	return int64(p), nil
}

func (r *mqlAlicloudRdsInstance) tags() (map[string]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RdsClient(r.region)
	if err != nil {
		return map[string]any{}, nil
	}
	resp, err := client.DescribeTags(&rdsclient.DescribeTagsRequest{
		RegionId:     tea.String(r.region),
		DBInstanceId: tea.String(r.instanceId),
	})
	if err != nil || resp == nil || resp.Body == nil || resp.Body.Items == nil {
		return map[string]any{}, nil
	}
	res := map[string]any{}
	for _, t := range resp.Body.Items.TagInfos {
		if t == nil || t.TagKey == nil {
			continue
		}
		res[*t.TagKey] = tea.StringValue(t.TagValue)
	}
	return res, nil
}

// fetchSSL lazily loads the per-instance DescribeDBInstanceSSL detail and
// caches it. Returns nil when the call fails, so callers fall back to their
// safe default.
func (r *mqlAlicloudRdsInstance) fetchSSL() *rdsclient.DescribeDBInstanceSSLResponseBody {
	r.sslLock.Lock()
	defer r.sslLock.Unlock()
	if r.sslDone {
		return r.ssl
	}
	r.sslDone = true

	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RdsClient(r.region)
	if err != nil {
		return nil
	}
	resp, err := client.DescribeDBInstanceSSL(&rdsclient.DescribeDBInstanceSSLRequest{
		DBInstanceId: tea.String(r.instanceId),
	})
	if err != nil || resp == nil || resp.Body == nil {
		return nil
	}
	r.ssl = resp.Body
	return r.ssl
}

func (r *mqlAlicloudRdsInstance) sslEnabled() (bool, error) {
	ssl := r.fetchSSL()
	if ssl == nil || ssl.SSLEnabled == nil {
		return false, nil
	}
	// MySQL/SQL Server report Yes/No, PostgreSQL reports on/off.
	switch strings.ToLower(strings.TrimSpace(*ssl.SSLEnabled)) {
	case "yes", "on", "1":
		return true, nil
	default:
		return false, nil
	}
}

func (r *mqlAlicloudRdsInstance) sslExpireTime() (*time.Time, error) {
	ssl := r.fetchSSL()
	if ssl == nil {
		return nil, nil
	}
	return rdsParseTime(ssl.SSLExpireTime), nil
}

func (r *mqlAlicloudRdsInstance) tdeEnabled() (bool, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RdsClient(r.region)
	if err != nil {
		return false, nil
	}
	resp, err := client.DescribeDBInstanceTDE(&rdsclient.DescribeDBInstanceTDERequest{
		DBInstanceId: tea.String(r.instanceId),
	})
	if err != nil || resp == nil || resp.Body == nil || resp.Body.TDEStatus == nil {
		return false, nil
	}
	return strings.EqualFold(strings.TrimSpace(*resp.Body.TDEStatus), "Enabled"), nil
}

func (r *mqlAlicloudRdsInstance) securityIPList() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RdsClient(r.region)
	if err != nil {
		return []any{}, nil
	}
	resp, err := client.DescribeDBInstanceIPArrayList(&rdsclient.DescribeDBInstanceIPArrayListRequest{
		DBInstanceId: tea.String(r.instanceId),
	})
	if err != nil || resp == nil || resp.Body == nil || resp.Body.Items == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, arr := range resp.Body.Items.DBInstanceIPArray {
		if arr == nil || arr.SecurityIPList == nil {
			continue
		}
		for _, ip := range strings.Split(*arr.SecurityIPList, ",") {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				res = append(res, ip)
			}
		}
	}
	return res, nil
}

func (r *mqlAlicloudRdsInstance) vpc() (*mqlAlicloudVpcNetwork, error) {
	if r.cacheVpcID == "" {
		r.Vpc.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return resolveVpcNetwork(r.MqlRuntime, r.cacheRegion, r.cacheVpcID)
}

func (r *mqlAlicloudRdsInstance) vswitch() (*mqlAlicloudVpcVswitch, error) {
	if r.cacheVswitchID == "" {
		r.Vswitch.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return resolveVpcVswitch(r.MqlRuntime, r.cacheRegion, r.cacheVswitchID)
}

func (r *mqlAlicloudRdsInstance) securityGroupIds() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RdsClient(r.region)
	if err != nil {
		return []any{}, nil
	}
	resp, err := client.DescribeSecurityGroupConfiguration(&rdsclient.DescribeSecurityGroupConfigurationRequest{
		DBInstanceId: tea.String(r.instanceId),
	})
	if err != nil || resp == nil || resp.Body == nil || resp.Body.Items == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, rel := range resp.Body.Items.EcsSecurityGroupRelation {
		if rel == nil || rel.SecurityGroupId == nil {
			continue
		}
		res = append(res, *rel.SecurityGroupId)
	}
	return res, nil
}
