// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
)

func TestFindParameterValue(t *testing.T) {
	params := []*sdk.Parameter{
		{Key: "NETWORK_POLICY", Value: "my_policy"},
		{Key: "ABORT_DETACHED_QUERY", Value: "false"},
		nil,
		{Key: "STATEMENT_TIMEOUT_IN_SECONDS", Value: "3600"},
	}

	cases := []struct {
		name string
		in   []*sdk.Parameter
		key  string
		want string
	}{
		{"exact match", params, "NETWORK_POLICY", "my_policy"},
		{"case-insensitive match", params, "network_policy", "my_policy"},
		{"other parameter", params, "STATEMENT_TIMEOUT_IN_SECONDS", "3600"},
		{"missing key returns empty", params, "DOES_NOT_EXIST", ""},
		{"empty value parameter", []*sdk.Parameter{{Key: "NETWORK_POLICY", Value: ""}}, "NETWORK_POLICY", ""},
		{"nil slice returns empty", nil, "NETWORK_POLICY", ""},
		{"slice with only nil entry", []*sdk.Parameter{nil}, "NETWORK_POLICY", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findParameterValue(tc.in, tc.key)
			if got != tc.want {
				t.Errorf("findParameterValue(%v, %q) = %q, want %q", tc.in, tc.key, got, tc.want)
			}
		})
	}
}
