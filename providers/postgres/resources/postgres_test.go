// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"reflect"
	"testing"
)

func TestClassifyPassword(t *testing.T) {
	scram := "SCRAM-SHA-256$4096:abc$def:ghi"
	md5 := "md5deadbeef"
	other := "plaintext"
	cases := []struct {
		name string
		in   *string
		want string
	}{
		{"nil", nil, "none"},
		{"empty", strPtr(""), "none"},
		{"scram", &scram, "scram-sha-256"},
		{"md5", &md5, "md5"},
		{"other", &other, "other"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyPassword(tc.in); got != tc.want {
				t.Errorf("classifyPassword = %q, want %q", got, tc.want)
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func TestSanitizeConnInfo(t *testing.T) {
	cases := map[string]string{
		"host=remote dbname=x password=secret user=u": "host=remote dbname=x password=REDACTED user=u",
		"host=remote user=u":                          "host=remote user=u",
		"password=secret":                             "password=REDACTED",
		// single-quoted value with a space must be fully redacted
		"host=remote password='s3cret value' user=u": "host=remote password=REDACTED user=u",
	}
	for in, want := range cases {
		if got := sanitizeConnInfo(in); got != want {
			t.Errorf("sanitizeConnInfo(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRedactOptions(t *testing.T) {
	got := redactOptions([]string{"user=remoteuser", "password=notreal", "PASSWORD=x", "host=remote"})
	want := []any{"user=remoteuser", "host=remote"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("redactOptions = %v, want %v", got, want)
	}
}

func TestIdentifierBuilders(t *testing.T) {
	if got := roleResourceID("SYS", "app_admin"); got != "app_admin@SYS" {
		t.Errorf("roleResourceID = %q", got)
	}
	if got := databaseResourceID("SYS", "appdb"); got != "SYS/appdb" {
		t.Errorf("databaseResourceID = %q", got)
	}
	// composite privilege id must vary by grantee and type
	a := privilegeResourceID("p", "PUBLIC", "CONNECT")
	b := privilegeResourceID("p", "PUBLIC", "TEMP")
	if a == b {
		t.Errorf("privilegeResourceID collides across type: %q", a)
	}
}
