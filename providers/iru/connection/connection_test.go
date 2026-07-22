// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "testing"

func TestNormalizeSubdomain(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"bare subdomain", "mondoo", "mondoo"},
		{"uppercased", "MonDoo", "mondoo"},
		{"surrounding whitespace", "  mondoo  ", "mondoo"},
		{"full host", "mondoo.api.kandji.io", "mondoo"},
		{"full https url", "https://mondoo.api.kandji.io", "mondoo"},
		{"full url with trailing slash", "https://mondoo.api.kandji.io/", "mondoo"},
		{"url with path and query", "https://mondoo.api.kandji.io/api/v1?x=1", "mondoo"},
		{"http scheme", "http://mondoo.api.kandji.io", "mondoo"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeSubdomain(tt.in); got != tt.want {
				t.Errorf("normalizeSubdomain(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestAPIURLFromSubdomain(t *testing.T) {
	if got, want := apiURLFromSubdomain("mondoo"), "https://mondoo.api.kandji.io"; got != want {
		t.Errorf("apiURLFromSubdomain(mondoo) = %q, want %q", got, want)
	}
}
