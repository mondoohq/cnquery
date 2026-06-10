// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestUrlUsesLdaps(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "ldaps lowercase", url: "ldaps://dc.example.com:636", want: true},
		{name: "ldaps uppercase", url: "LDAPS://dc.example.com:636", want: true},
		{name: "ldaps mixed case", url: "LdApS://dc.example.com:636", want: true},
		{name: "ldap insecure", url: "ldap://dc.example.com:389", want: false},
		{name: "empty", url: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := urlUsesLdaps(tt.url); got != tt.want {
				t.Errorf("urlUsesLdaps(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
