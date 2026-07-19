// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/alibabacloud-go/tea/tea"
	wafclient "github.com/alibabacloud-go/waf-openapi-20211001/v7/client"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlAlicloudWaf) id() (string, error) {
	return "alicloud.waf", nil
}

func (r *mqlAlicloudWaf) instances() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)

	res := []any{}
	// WAF is a center service; an account lives in one partition, so a call
	// against the other center errors. Try both and skip a failing center, but
	// remember the error and surface it only if NEITHER center responded, so a
	// transient outage is not silently reported as "no WAF".
	var lastErr error
	succeeded := 0
	for _, region := range alicloudCenterRegions {
		client, err := conn.WafClient(region)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := client.DescribeInstance(&wafclient.DescribeInstanceRequest{
			RegionId: tea.String(region),
		})
		if err != nil {
			lastErr = err
			continue
		}
		succeeded++
		if resp == nil || resp.Body == nil || resp.Body.InstanceId == nil || *resp.Body.InstanceId == "" {
			// this center responded but the account has no WAF instance here
			continue
		}
		mqlInstance, err := newWafInstance(r.MqlRuntime, region, resp.Body)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}
	if succeeded == 0 && lastErr != nil {
		return nil, lastErr
	}
	return res, nil
}

// mqlAlicloudWafInstanceInternal caches the center region and instance id used
// by the child accessors.
type mqlAlicloudWafInstanceInternal struct {
	region     string
	instanceId string
}

func newWafInstance(runtime *plugin.Runtime, region string, body *wafclient.DescribeInstanceResponseBody) (*mqlAlicloudWafInstance, error) {
	instanceID := tea.StringValue(body.InstanceId)
	resource, err := CreateResource(runtime, "alicloud.waf.instance", map[string]*llx.RawData{
		"__id":       llx.StringData(region + "/" + instanceID),
		"regionId":   llx.StringData(region),
		"instanceId": llx.StringData(instanceID),
		"status":     llx.IntData(int64(tea.Int32Value(body.Status))),
		"edition":    llx.StringDataPtr(body.Edition),
		"payType":    llx.StringDataPtr(body.PayType),
		"inDebt":     llx.StringDataPtr(body.InDebt),
		"startTime":  llx.TimeDataPtr(configEpochMillis(body.StartTime)),
		"endTime":    llx.TimeDataPtr(configEpochMillis(body.EndTime)),
	})
	if err != nil {
		return nil, err
	}
	mqlInstance := resource.(*mqlAlicloudWafInstance)
	mqlInstance.region = region
	mqlInstance.instanceId = instanceID
	return mqlInstance, nil
}

func (r *mqlAlicloudWafInstance) id() (string, error) {
	return r.region + "/" + r.instanceId, nil
}

func (r *mqlAlicloudWafInstance) defenseResources() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.WafClient(r.region)
	if err != nil {
		return nil, err
	}

	res := []any{}
	pageNumber := int32(1)
	pageSize := int32(100)
	for {
		resp, err := client.DescribeDefenseResources(&wafclient.DescribeDefenseResourcesRequest{
			InstanceId: tea.String(r.instanceId),
			RegionId:   tea.String(r.region),
			PageNumber: tea.Int32(pageNumber),
			PageSize:   tea.Int32(pageSize),
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil {
			break
		}
		items := resp.Body.Resources
		for _, dr := range items {
			if dr == nil || dr.Resource == nil {
				continue
			}
			resource, err := CreateResource(r.MqlRuntime, "alicloud.waf.defenseResource", map[string]*llx.RawData{
				"__id":           llx.StringData(r.region + "/" + r.instanceId + "/" + tea.StringValue(dr.Resource)),
				"regionId":       llx.StringData(r.region),
				"instanceId":     llx.StringData(r.instanceId),
				"resource":       llx.StringDataPtr(dr.Resource),
				"product":        llx.StringDataPtr(dr.Product),
				"pattern":        llx.StringDataPtr(dr.Pattern),
				"resourceStatus": llx.StringDataPtr(dr.ResourceStatus),
				"resourceGroup":  llx.StringDataPtr(dr.ResourceGroup),
				"createTime":     llx.TimeDataPtr(configEpochMillis(dr.GmtCreate)),
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

func (r *mqlAlicloudWafInstance) domains() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.WafClient(r.region)
	if err != nil {
		return nil, err
	}

	res := []any{}
	pageNumber := int64(1)
	pageSize := int64(100)
	for {
		resp, err := client.DescribeDomains(&wafclient.DescribeDomainsRequest{
			InstanceId: tea.String(r.instanceId),
			RegionId:   tea.String(r.region),
			PageNumber: tea.Int64(pageNumber),
			PageSize:   tea.Int64(pageSize),
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil {
			break
		}
		items := resp.Body.Domains
		for _, d := range items {
			if d == nil || d.Domain == nil {
				continue
			}
			mqlDomain, err := newWafDomain(r.MqlRuntime, r.region, r.instanceId, d)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlDomain)
		}
		if len(items) < int(pageSize) {
			break
		}
		pageNumber++
	}
	return res, nil
}

// mqlAlicloudWafDomainInternal caches the keys needed to fetch the per-domain
// TLS detail and memoizes it.
type mqlAlicloudWafDomainInternal struct {
	region     string
	instanceId string
	domain     string

	detailLock    sync.Mutex
	detailFetched atomic.Bool
	detail        *wafclient.DescribeDomainDetailResponseBody
}

func newWafDomain(runtime *plugin.Runtime, region, instanceID string, d *wafclient.DescribeDomainsResponseBodyDomains) (*mqlAlicloudWafDomain, error) {
	httpPorts := []any{}
	httpsPorts := []any{}
	if d.ListenPorts != nil {
		httpPorts = int64PtrsToInts(d.ListenPorts.Http)
		httpsPorts = int64PtrsToInts(d.ListenPorts.Https)
	}

	resource, err := CreateResource(runtime, "alicloud.waf.domain", map[string]*llx.RawData{
		"__id":         llx.StringData(region + "/" + instanceID + "/" + tea.StringValue(d.Domain)),
		"regionId":     llx.StringData(region),
		"instanceId":   llx.StringData(instanceID),
		"domain":       llx.StringDataPtr(d.Domain),
		"cname":        llx.StringDataPtr(d.Cname),
		"status":       llx.IntData(int64(tea.Int32Value(d.Status))),
		"httpPorts":    llx.ArrayData(httpPorts, types.Int),
		"httpsPorts":   llx.ArrayData(httpsPorts, types.Int),
		"httpsEnabled": llx.BoolData(len(httpsPorts) > 0),
	})
	if err != nil {
		return nil, err
	}
	mqlDomain := resource.(*mqlAlicloudWafDomain)
	mqlDomain.region = region
	mqlDomain.instanceId = instanceID
	mqlDomain.domain = tea.StringValue(d.Domain)
	return mqlDomain, nil
}

func (r *mqlAlicloudWafDomain) id() (string, error) {
	return r.region + "/" + r.instanceId + "/" + r.domain, nil
}

// domainDetail lazily fetches DescribeDomainDetail for the TLS configuration. A
// transient error is not cached and is returned.
func (r *mqlAlicloudWafDomain) domainDetail() (*wafclient.DescribeDomainDetailResponseBody, error) {
	if r.detailFetched.Load() {
		return r.detail, nil
	}
	r.detailLock.Lock()
	defer r.detailLock.Unlock()
	if r.detailFetched.Load() {
		return r.detail, nil
	}

	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.WafClient(r.region)
	if err != nil {
		return nil, err
	}
	resp, err := client.DescribeDomainDetail(&wafclient.DescribeDomainDetailRequest{
		InstanceId: tea.String(r.instanceId),
		RegionId:   tea.String(r.region),
		Domain:     tea.String(r.domain),
	})
	if err != nil {
		return nil, err
	}
	if resp != nil {
		r.detail = resp.Body
	}
	r.detailFetched.Store(true)
	return r.detail, nil
}

func (r *mqlAlicloudWafDomain) certId() (string, error) {
	d, err := r.domainDetail()
	if err != nil || d == nil || d.Listen == nil {
		return "", err
	}
	return tea.StringValue(d.Listen.CertId), nil
}

func (r *mqlAlicloudWafDomain) tlsVersion() (string, error) {
	d, err := r.domainDetail()
	if err != nil || d == nil || d.Listen == nil {
		return "", err
	}
	return tea.StringValue(d.Listen.TLSVersion), nil
}

func (r *mqlAlicloudWafDomain) tls13Enabled() (bool, error) {
	d, err := r.domainDetail()
	if err != nil || d == nil || d.Listen == nil {
		return false, err
	}
	return tea.BoolValue(d.Listen.EnableTLSv3), nil
}

func (r *mqlAlicloudWafDomain) certExpireTime() (*time.Time, error) {
	d, err := r.domainDetail()
	if err != nil || d == nil || d.CertDetail == nil {
		return nil, err
	}
	return configEpochMillis(d.CertDetail.EndTime), nil
}
