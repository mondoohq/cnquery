// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"reflect"
	"testing"
)

func TestParseAuthPolicyList(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []any
	}{
		{"empty string", "", []any{}},
		{"empty parens", "()", []any{}},
		{"single quoted", "('ALL')", []any{"ALL"}},
		{"multiple quoted", "('PASSWORD', 'SAML')", []any{"PASSWORD", "SAML"}},
		{"no parens", "ALL", []any{"ALL"}},
		{"double-quoted", `("PASSWORD","SAML")`, []any{"PASSWORD", "SAML"}},
		{"extra whitespace", "(  'PASSWORD' , 'SAML'  )", []any{"PASSWORD", "SAML"}},
		{"trailing comma yields no empty entry", "('PASSWORD',)", []any{"PASSWORD"}},
		// Snowflake actually returns these lists in square brackets, unquoted.
		{"bracket single", "[ALL]", []any{"ALL"}},
		{"bracket multiple", "[PASSWORD, SAML]", []any{"PASSWORD", "SAML"}},
		{"bracket empty", "[]", []any{}},
		{"bracket no space", "[PASSWORD,SAML]", []any{"PASSWORD", "SAML"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseAuthPolicyList(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseAuthPolicyList(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
