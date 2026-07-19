// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"sync"
	"time"

	configclient "github.com/alibabacloud-go/config-20200907/v4/client"
	tea "github.com/alibabacloud-go/tea/tea"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
	"go.mondoo.com/mql/v13/types"
)

// configEpochMillis converts an epoch-milliseconds timestamp into a *time.Time,
// returning nil when the value is nil or zero.
func configEpochMillis(v *int64) *time.Time {
	if v == nil || *v == 0 {
		return nil
	}
	t := time.UnixMilli(*v).UTC()
	return &t
}

// mqlAlicloudConfigInternal memoizes the configuration-recorder detail shared by
// the recorderEnabled, recorderStatus, and recordedResourceTypes accessors.
type mqlAlicloudConfigInternal struct {
	recorderOnce sync.Once
	recorder     *configclient.GetConfigurationRecorderResponseBodyConfigurationRecorder
}

func (r *mqlAlicloudConfig) id() (string, error) {
	return "alicloud.config", nil
}

func (r *mqlAlicloudConfig) rules() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.ConfigClient()
	if err != nil {
		return nil, err
	}

	res := []any{}
	pageNumber := int32(1)
	pageSize := int32(100)
	for {
		resp, err := client.ListConfigRules(&configclient.ListConfigRulesRequest{
			PageNumber: tea.Int32(pageNumber),
			PageSize:   tea.Int32(pageSize),
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil || resp.Body.ConfigRules == nil {
			break
		}

		items := resp.Body.ConfigRules.ConfigRuleList
		for _, rule := range items {
			if rule == nil || rule.ConfigRuleId == nil {
				continue
			}
			mqlRule, err := newConfigRule(r.MqlRuntime, rule)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRule)
		}

		total := tea.Int64Value(resp.Body.ConfigRules.TotalCount)
		if len(items) == 0 || int64(pageNumber)*int64(pageSize) >= total {
			break
		}
		pageNumber++
	}
	return res, nil
}

// newConfigRule builds a fully populated alicloud.config.rule from a
// ListConfigRules item.
func newConfigRule(runtime *plugin.Runtime, rule *configclient.ListConfigRulesResponseBodyConfigRulesConfigRuleList) (*mqlAlicloudConfigRule, error) {
	resourceTypes := []any{}
	if rule.ResourceTypesScope != nil {
		for _, t := range strings.Split(*rule.ResourceTypesScope, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				resourceTypes = append(resourceTypes, t)
			}
		}
	}

	complianceType := ""
	complianceCount := int64(0)
	if rule.Compliance != nil {
		complianceType = tea.StringValue(rule.Compliance.ComplianceType)
		complianceCount = int64(tea.Int32Value(rule.Compliance.Count))
	}

	compliancePackId := ""
	if rule.CreateBy != nil {
		compliancePackId = tea.StringValue(rule.CreateBy.CompliancePackId)
	}

	tags := map[string]any{}
	for _, t := range rule.Tags {
		if t == nil || t.Key == nil {
			continue
		}
		tags[*t.Key] = tea.StringValue(t.Value)
	}

	resource, err := CreateResource(runtime, "alicloud.config.rule", map[string]*llx.RawData{
		"__id":               llx.StringDataPtr(rule.ConfigRuleId),
		"configRuleId":       llx.StringDataPtr(rule.ConfigRuleId),
		"configRuleName":     llx.StringDataPtr(rule.ConfigRuleName),
		"configRuleArn":      llx.StringDataPtr(rule.ConfigRuleArn),
		"configRuleState":    llx.StringDataPtr(rule.ConfigRuleState),
		"description":        llx.StringDataPtr(rule.Description),
		"riskLevel":          llx.IntData(int64(tea.Int32Value(rule.RiskLevel))),
		"sourceOwner":        llx.StringDataPtr(rule.SourceOwner),
		"sourceIdentifier":   llx.StringDataPtr(rule.SourceIdentifier),
		"automationType":     llx.StringDataPtr(rule.AutomationType),
		"resourceTypesScope": llx.ArrayData(resourceTypes, types.String),
		"complianceType":     llx.StringData(complianceType),
		"complianceCount":    llx.IntData(complianceCount),
		"compliancePackId":   llx.StringData(compliancePackId),
		"tags":               llx.MapData(tags, types.String),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAlicloudConfigRule), nil
}

// recorderDetail lazily fetches and caches the configuration recorder.
func (r *mqlAlicloudConfig) recorderDetail() *configclient.GetConfigurationRecorderResponseBodyConfigurationRecorder {
	r.recorderOnce.Do(func() {
		conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
		client, err := conn.ConfigClient()
		if err != nil {
			return
		}
		resp, err := client.GetConfigurationRecorder()
		if err != nil || resp == nil || resp.Body == nil {
			return
		}
		r.recorder = resp.Body.ConfigurationRecorder
	})
	return r.recorder
}

func (r *mqlAlicloudConfig) recorderStatus() (string, error) {
	rec := r.recorderDetail()
	if rec == nil {
		return "", nil
	}
	return tea.StringValue(rec.ConfigurationRecorderStatus), nil
}

func (r *mqlAlicloudConfig) recorderEnabled() (bool, error) {
	rec := r.recorderDetail()
	if rec == nil {
		return false, nil
	}
	return tea.StringValue(rec.ConfigurationRecorderStatus) == "REGISTERED", nil
}

func (r *mqlAlicloudConfig) recordedResourceTypes() ([]any, error) {
	rec := r.recorderDetail()
	if rec == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, t := range rec.ResourceTypes {
		if t == nil {
			continue
		}
		res = append(res, tea.StringValue(t))
	}
	return res, nil
}

func (r *mqlAlicloudConfig) complianceSummary() (any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.ConfigClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetComplianceSummary()
	if err != nil || resp == nil || resp.Body == nil || resp.Body.ComplianceSummary == nil {
		return nil, nil
	}
	summary := resp.Body.ComplianceSummary

	res := map[string]any{}
	if byRule := summary.ComplianceSummaryByConfigRule; byRule != nil {
		res["compliantRuleCount"] = int64(tea.Int32Value(byRule.CompliantCount))
		res["nonCompliantRuleCount"] = int64(tea.Int32Value(byRule.NonCompliantCount))
		res["totalRuleCount"] = tea.Int64Value(byRule.TotalCount)
	}
	if byResource := summary.ComplianceSummaryByResource; byResource != nil {
		res["compliantResourceCount"] = int64(tea.Int32Value(byResource.CompliantCount))
		res["nonCompliantResourceCount"] = int64(tea.Int32Value(byResource.NonCompliantCount))
		res["totalResourceCount"] = tea.Int64Value(byResource.TotalCount)
		res["highRiskNonCompliantResourceCount"] = int64(tea.Int32Value(byResource.HighRiskRuleNonCompliantResourceCount))
		res["mediumRiskNonCompliantResourceCount"] = int64(tea.Int32Value(byResource.MediumRiskRuleNonCompliantResourceCount))
		res["lowRiskNonCompliantResourceCount"] = int64(tea.Int32Value(byResource.LowRiskRuleNonCompliantResourceCount))
	}
	return res, nil
}

func (r *mqlAlicloudConfig) deliveryChannels() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.ConfigClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListConfigDeliveryChannels(&configclient.ListConfigDeliveryChannelsRequest{})
	if err != nil || resp == nil || resp.Body == nil {
		return []any{}, nil
	}

	res := []any{}
	for _, ch := range resp.Body.DeliveryChannels {
		if ch == nil {
			continue
		}
		res = append(res, map[string]any{
			"channelId":                           tea.StringValue(ch.DeliveryChannelId),
			"name":                                tea.StringValue(ch.DeliveryChannelName),
			"type":                                tea.StringValue(ch.DeliveryChannelType),
			"targetArn":                           tea.StringValue(ch.DeliveryChannelTargetArn),
			"assumeRoleArn":                       tea.StringValue(ch.DeliveryChannelAssumeRoleArn),
			"enabled":                             tea.Int32Value(ch.Status) == 1,
			"description":                         tea.StringValue(ch.Description),
			"configurationItemChangeNotification": tea.BoolValue(ch.ConfigurationItemChangeNotification),
			"configurationSnapshot":               tea.BoolValue(ch.ConfigurationSnapshot),
			"compliantSnapshot":                   tea.BoolValue(ch.CompliantSnapshot),
			"nonCompliantNotification":            tea.BoolValue(ch.NonCompliantNotification),
		})
	}
	return res, nil
}

// mqlAlicloudConfigRuleInternal memoizes the GetConfigRule detail shared by the
// timestamp and execution-frequency accessors.
type mqlAlicloudConfigRuleInternal struct {
	detailOnce sync.Once
	detail     *configclient.GetConfigRuleResponseBodyConfigRule
}

func (r *mqlAlicloudConfigRule) id() (string, error) {
	return r.ConfigRuleId.Data, nil
}

// detailFor lazily fetches and caches the GetConfigRule detail.
func (r *mqlAlicloudConfigRule) detailFor() *configclient.GetConfigRuleResponseBodyConfigRule {
	r.detailOnce.Do(func() {
		conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
		client, err := conn.ConfigClient()
		if err != nil {
			return
		}
		resp, err := client.GetConfigRule(&configclient.GetConfigRuleRequest{
			ConfigRuleId: tea.String(r.ConfigRuleId.Data),
		})
		if err != nil || resp == nil || resp.Body == nil {
			return
		}
		r.detail = resp.Body.ConfigRule
	})
	return r.detail
}

func (r *mqlAlicloudConfigRule) maximumExecutionFrequency() (string, error) {
	d := r.detailFor()
	if d == nil {
		return "", nil
	}
	return tea.StringValue(d.MaximumExecutionFrequency), nil
}

func (r *mqlAlicloudConfigRule) createTime() (*time.Time, error) {
	d := r.detailFor()
	if d == nil {
		return nil, nil
	}
	return configEpochMillis(d.CreateTimestamp), nil
}

func (r *mqlAlicloudConfigRule) modifiedTime() (*time.Time, error) {
	d := r.detailFor()
	if d == nil {
		return nil, nil
	}
	return configEpochMillis(d.ModifiedTimestamp), nil
}
