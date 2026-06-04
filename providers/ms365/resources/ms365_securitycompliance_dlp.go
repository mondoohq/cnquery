// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// dlpPolicies returns the Data Loss Prevention compliance policies as typed
// resources. The underlying data comes from the Security & Compliance
// PowerShell report, whose JSON shape varies by module version, so each field
// is extracted defensively rather than struct-decoded.
func (r *mqlMs365ExchangeonlineSecurityAndCompliance) dlpPolicies() ([]any, error) {
	report, err := r.getSecurityAndComplianceReport()
	if err != nil {
		return nil, err
	}
	return convertDlpCompliancePolicies(r.MqlRuntime, report.DlpCompliancePolicy)
}

func convertDlpCompliancePolicies(runtime *plugin.Runtime, raw []any) ([]any, error) {
	result := []any{}
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		guid := dlpString(m, "Guid")
		id := guid
		if id == "" {
			id = dlpString(m, "Name")
		}

		mql, err := CreateResource(runtime, "ms365.exchangeonline.dlpCompliancePolicy",
			map[string]*llx.RawData{
				"__id":               llx.StringData("dlpCompliancePolicy-" + id),
				"name":               llx.StringData(dlpString(m, "Name")),
				"guid":               llx.StringData(guid),
				"mode":               llx.StringData(dlpString(m, "Mode")),
				"enabled":            llx.BoolData(dlpBool(m, "Enabled")),
				"workload":           llx.StringData(dlpWorkload(m)),
				"priority":           llx.IntData(dlpInt(m, "Priority")),
				"comment":            llx.StringData(dlpString(m, "Comment")),
				"createdBy":          llx.StringData(dlpString(m, "CreatedBy")),
				"isValid":            llx.BoolData(dlpBool(m, "IsValid")),
				"distributionStatus": llx.StringData(dlpString(m, "DistributionStatus")),
				"whenCreated":        llx.TimeDataPtr(dlpTime(m, "WhenCreated")),
				"whenChanged":        llx.TimeDataPtr(dlpTime(m, "WhenChanged")),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

func dlpString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func dlpBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func dlpInt(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

// dlpWorkload coerces the Workload property, which may serialize as a single
// string, a list of workload names, or (for a numeric enum) be unreadable.
func dlpWorkload(m map[string]any) string {
	switch v := m["Workload"].(type) {
	case string:
		return v
	case []any:
		parts := []string{}
		for _, p := range v {
			if s, ok := p.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	}
	return ""
}

// dlpTime parses a timestamp that PowerShell may emit either as an ISO 8601
// string or as the legacy "/Date(milliseconds)/" form.
func dlpTime(m map[string]any, key string) *time.Time {
	s, ok := m[key].(string)
	if !ok || s == "" {
		return nil
	}

	if strings.HasPrefix(s, "/Date(") && strings.HasSuffix(s, ")/") {
		inner := strings.TrimSuffix(strings.TrimPrefix(s, "/Date("), ")/")
		if idx := strings.IndexAny(inner, "+-"); idx > 0 {
			inner = inner[:idx]
		}
		if ms, err := strconv.ParseInt(inner, 10, 64); err == nil {
			t := time.UnixMilli(ms).UTC()
			return &t
		}
		return nil
	}

	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
