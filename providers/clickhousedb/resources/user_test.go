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

// TestStringList covers the two shapes system.users.auth_type actually takes:
// a scalar Enum8 up to the 24.8 LTS line, and an Array(Enum8) from 25.x. The
// error cases matter as much as the happy ones -- an unrecognised shape read as
// an empty list would make requiresCredential report a password-less account as
// requiring a credential.
func TestStringList(t *testing.T) {
	cases := []struct {
		name    string
		in      any
		want    []string
		wantErr bool
	}{
		{"array, as 25.x returns it", []string{"sha256_password"}, []string{"sha256_password"}, false},
		{"array with several methods", []string{"sha256_password", "kerberos"}, []string{"sha256_password", "kerberos"}, false},
		{"scalar, as 24.8 returns it", "plaintext_password", []string{"plaintext_password"}, false},
		{"empty scalar is no entry", "", nil, false},
		{"nil column", nil, nil, false},
		{"driver any-slice", []any{"no_password"}, []string{"no_password"}, false},
		{"empty array stays empty", []string{}, []string{}, false},
		{"unknown type errors rather than reading as empty", 42, nil, true},
		{"mixed any-slice errors", []any{"ldap", 7}, nil, true},
	}
	for _, c := range cases {
		got, err := stringList("auth_type", c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: stringList(%v) error = %v, wantErr %v", c.name, c.in, err, c.wantErr)
			continue
		}
		if c.wantErr {
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: stringList(%v) = %v, want %v", c.name, c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: stringList(%v) = %v, want %v", c.name, c.in, got, c.want)
				break
			}
		}
	}
}
