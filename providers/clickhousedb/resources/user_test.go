// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestRequiresCredential(t *testing.T) {
	cases := []struct {
		name      string
		authTypes []string
		want      bool
	}{
		{"password only", []string{"sha256_password"}, true},
		{"no_password only", []string{"no_password"}, false},
		// ClickHouse admits a login if any method matches, so a no_password entry
		// alongside a real credential still means the account is reachable without one.
		{"mixed with no_password", []string{"sha256_password", "no_password"}, false},
		{"multiple real methods", []string{"sha256_password", "ldap"}, true},
		// An empty method list is anomalous; treated defensively as requiring a credential.
		{"empty", nil, true},
	}
	for _, c := range cases {
		if got := requiresCredential(c.authTypes); got != c.want {
			t.Errorf("%s: requiresCredential(%v) = %v, want %v", c.name, c.authTypes, got, c.want)
		}
	}
}

func TestAllowsAnyHost(t *testing.T) {
	cases := []struct {
		hostIps []string
		want    bool
	}{
		{[]string{"::/0"}, true},
		{[]string{"0.0.0.0/0"}, true},
		{[]string{"10.0.0.0/8"}, false},
		{[]string{"10.0.0.0/8", "::/0"}, true},
		{nil, false},
	}
	for _, c := range cases {
		if got := allowsAnyHost(c.hostIps); got != c.want {
			t.Errorf("allowsAnyHost(%v) = %v, want %v", c.hostIps, got, c.want)
		}
	}
}
