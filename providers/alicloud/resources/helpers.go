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
