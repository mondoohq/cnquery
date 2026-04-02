// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestDnToDomain(t *testing.T) {
	tests := []struct {
		dn   string
		want string
	}{
		{"DC=corp,DC=example,DC=com", "corp.example.com"},
		{"DC=example,DC=com", "example.com"},
		{"DC=single", "single"},
		{"CN=Users,DC=corp,DC=example,DC=com", "corp.example.com"},
		{"", ""},
		{"CN=SomeObject", ""},
	}

	for _, tt := range tests {
		t.Run(tt.dn, func(t *testing.T) {
			got := dnToDomain(tt.dn)
			if got != tt.want {
				t.Errorf("dnToDomain(%q) = %q, want %q", tt.dn, got, tt.want)
			}
		})
	}
}

func TestExtractSiteFromServerRef(t *testing.T) {
	tests := []struct {
		name string
		dn   string
		want string
	}{
		{
			name: "standard server reference",
			dn:   "CN=DC01,CN=Servers,CN=Default-First-Site-Name,CN=Sites,CN=Configuration,DC=corp,DC=example,DC=com",
			want: "Default-First-Site-Name",
		},
		{
			name: "different site",
			dn:   "CN=DC02,CN=Servers,CN=NewYork,CN=Sites,CN=Configuration,DC=corp,DC=com",
			want: "NewYork",
		},
		{
			name: "empty string",
			dn:   "",
			want: "",
		},
		{
			name: "no site structure",
			dn:   "CN=DC01,OU=Domain Controllers,DC=corp,DC=com",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSiteFromServerRef(tt.dn)
			if got != tt.want {
				t.Errorf("extractSiteFromServerRef(%q) = %q, want %q", tt.dn, got, tt.want)
			}
		})
	}
}

func TestDurationToDaysResource(t *testing.T) {
	tests := []struct {
		name string
		raw  int64
		want int64
	}{
		{"zero (no limit)", 0, 0},
		// AD stores -42 days as -(42 * 86400 * 10_000_000) = -36288000000000
		{"-42 days", -36288000000000, 42},
		{"-1 day", -864000000000, 1},
		// Positive value (shouldn't happen in practice, but handle gracefully)
		{"positive 1 day", 864000000000, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := durationToDays(tt.raw)
			if got != tt.want {
				t.Errorf("durationToDays(%d) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}
