// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import tea "github.com/alibabacloud-go/tea/tea"

// strPtrsToStrings converts a []*string SDK slice into a []string, dropping nil
// and empty entries so downstream resolvers are never handed a blank id.
func strPtrsToStrings(in []*string) []string {
	res := []string{}
	for _, s := range in {
		if v := tea.StringValue(s); v != "" {
			res = append(res, v)
		}
	}
	return res
}

// strPtrsToAny converts a []*string SDK slice into a []any of the non-empty
// strings, for populating MQL list fields.
func strPtrsToAny(in []*string) []any {
	res := []any{}
	for _, s := range in {
		if v := tea.StringValue(s); v != "" {
			res = append(res, v)
		}
	}
	return res
}

// int64PtrsToInts converts a []*int64 SDK slice into a []any of the non-nil
// values, for populating MQL int list fields.
func int64PtrsToInts(in []*int64) []any {
	res := []any{}
	for _, v := range in {
		if v != nil {
			res = append(res, *v)
		}
	}
	return res
}

// alicloudCenterRegions are the two center endpoints that WAF, Cloud Firewall,
// and Anti-DDoS answer at: cn-hangzhou for the China partition and
// ap-southeast-1 for the international partition. An account belongs to one of
// them, so a call against the other partition returns no data (or an error).
var alicloudCenterRegions = []string{"cn-hangzhou", "ap-southeast-1"}
