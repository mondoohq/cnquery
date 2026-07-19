// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"sync"
	"sync/atomic"

	cloudfwclient "github.com/alibabacloud-go/cloudfw-20171207/v8/client"
	tea "github.com/alibabacloud-go/tea/tea"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
)

// mqlAlicloudCloudFirewallInternal memoizes the center region the account's
// Cloud Firewall answers at (cn-hangzhou or ap-southeast-1) and the edition
// probe, so enabled/edition/controlPolicies share one lookup.
type mqlAlicloudCloudFirewallInternal struct {
	lock    sync.Mutex
	fetched atomic.Bool
	region  string
	version *cloudfwclient.DescribeUserBuyVersionResponseBody
}

func (r *mqlAlicloudCloudFirewall) id() (string, error) {
	return "alicloud.cloudFirewall", nil
}

// buyVersion probes the two centers for the account's Cloud Firewall edition and
// caches the working center. It returns the last error when NEITHER center
// responds, so a transient outage is surfaced rather than masked as "not
// provisioned". A success is cached; a total failure is not, so a later call
// retries.
func (r *mqlAlicloudCloudFirewall) buyVersion() (string, *cloudfwclient.DescribeUserBuyVersionResponseBody, error) {
	if r.fetched.Load() {
		return r.region, r.version, nil
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.fetched.Load() {
		return r.region, r.version, nil
	}

	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	var lastErr error
	for _, region := range alicloudCenterRegions {
		client, err := conn.CloudfwClient(region)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := client.DescribeUserBuyVersion(&cloudfwclient.DescribeUserBuyVersionRequest{})
		if err != nil {
			lastErr = err
			continue
		}
		if resp == nil || resp.Body == nil {
			continue
		}
		r.region = region
		r.version = resp.Body
		r.fetched.Store(true)
		return r.region, r.version, nil
	}
	return "", nil, lastErr
}

func (r *mqlAlicloudCloudFirewall) enabled() (bool, error) {
	_, v, err := r.buyVersion()
	if err != nil || v == nil {
		return false, err
	}
	return tea.BoolValue(v.UserStatus), nil
}

func (r *mqlAlicloudCloudFirewall) edition() (int64, error) {
	_, v, err := r.buyVersion()
	if err != nil || v == nil {
		return 0, err
	}
	return int64(tea.Int32Value(v.Version)), nil
}

func (r *mqlAlicloudCloudFirewall) controlPolicies() ([]any, error) {
	region, v, err := r.buyVersion()
	if err != nil {
		return nil, err
	}
	if region == "" || v == nil {
		// Cloud Firewall is not provisioned for this account
		return []any{}, nil
	}
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.CloudfwClient(region)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, direction := range []string{"in", "out"} {
		currentPage := 1
		for {
			resp, err := client.DescribeControlPolicy(&cloudfwclient.DescribeControlPolicyRequest{
				Direction:   tea.String(direction),
				CurrentPage: tea.String(strconv.Itoa(currentPage)),
				PageSize:    tea.String("100"),
			})
			if err != nil {
				return nil, err
			}
			if resp == nil || resp.Body == nil {
				break
			}
			items := resp.Body.Policys
			for _, p := range items {
				if p == nil || p.AclUuid == nil {
					continue
				}
				resource, err := CreateResource(r.MqlRuntime, "alicloud.cloudFirewall.controlPolicy", map[string]*llx.RawData{
					"__id":            llx.StringDataPtr(p.AclUuid),
					"aclUuid":         llx.StringDataPtr(p.AclUuid),
					"direction":       llx.StringDataPtr(p.Direction),
					"action":          llx.StringDataPtr(p.AclAction),
					"source":          llx.StringDataPtr(p.Source),
					"sourceType":      llx.StringDataPtr(p.SourceType),
					"destination":     llx.StringDataPtr(p.Destination),
					"destinationType": llx.StringDataPtr(p.DestinationType),
					"destPort":        llx.StringDataPtr(p.DestPort),
					"proto":           llx.StringDataPtr(p.Proto),
					"applicationName": llx.StringDataPtr(p.ApplicationName),
					"description":     llx.StringDataPtr(p.Description),
					"enabled":         llx.BoolData(tea.StringValue(p.Release) == "true"),
					"order":           llx.IntData(int64(tea.Int32Value(p.Order))),
					"hitTimes":        llx.IntData(tea.Int64Value(p.HitTimes)),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, resource)
			}
			if len(items) < 100 {
				break
			}
			currentPage++
		}
	}
	return res, nil
}

func (r *mqlAlicloudCloudFirewallControlPolicy) id() (string, error) {
	return r.AclUuid.Data, nil
}
