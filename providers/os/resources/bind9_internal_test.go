// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/resources/bind9"
)

func TestBind9ResolveZonePath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		directory  string
		declaredIn string
		expected   string
	}{
		{
			// The Debian default: a directory option plus relative zone files.
			name:       "relative path resolves against the directory option",
			path:       "db.example.com",
			directory:  "/var/cache/bind",
			declaredIn: "/etc/bind/named.conf.local",
			expected:   "/var/cache/bind/db.example.com",
		},
		{
			// Without a directory option, named uses its working directory.
			// Reading the file next to the declaration is the only answer
			// available from the configuration alone, and is where these files
			// sit in practice.
			name:       "relative path falls back to the declaring file",
			path:       "db.local",
			directory:  "",
			declaredIn: "/etc/bind/named.conf.default-zones",
			expected:   "/etc/bind/db.local",
		},
		{
			name:       "an absolute path is left alone",
			path:       "/usr/share/dns/root.hints",
			directory:  "/var/cache/bind",
			declaredIn: "/etc/bind/named.conf.default-zones",
			expected:   "/usr/share/dns/root.hints",
		},
		{
			// A forward or stub zone declares no file at all.
			name:     "no file yields no path",
			path:     "",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, bind9ResolveZonePath(test.path, test.directory, test.declaredIn))
		})
	}
}

func TestBind9IsYes(t *testing.T) {
	for _, v := range []string{"yes", "YES", "Yes", "true", "1", " yes "} {
		assert.True(t, bind9IsYes(v), "%q", v)
	}
	for _, v := range []string{"no", "NO", "false", "0", "", "auto", "maybe"} {
		assert.False(t, bind9IsYes(v), "%q", v)
	}
}

func TestBind9EachZone(t *testing.T) {
	// Zones live at the top level and inside views. A reader that only looks at
	// the top level reports a split-horizon server as serving nothing.
	stmts, err := bind9.Parse(`
zone "." { type hint; file "/usr/share/dns/root.hints"; };
view "internal" {
	match-clients { 10.0.0.0/8; };
	zone "corp.example" { type master; file "db.corp"; };
	zone "0.10.in-addr.arpa" { type master; file "db.10"; };
};
view "external" {
	zone "example.com" { type master; file "db.example.com"; };
};
`)
	require.NoError(t, err)

	type seen struct{ view, zone string }
	var got []seen
	bind9EachZone(stmts, func(view string, z bind9.Statement) {
		got = append(got, seen{view, z.Arg(0)})
	})

	assert.Equal(t, []seen{
		{"", "."},
		{"internal", "corp.example"},
		{"internal", "0.10.in-addr.arpa"},
		{"external", "example.com"},
	}, got)
}

func TestBind9EachZoneIgnoresNonZoneStatements(t *testing.T) {
	// A view carries more than zones, and a top-level acl or key is not one.
	stmts, err := bind9.Parse(`
acl "trusted" { 10.0.0.0/8; };
key "rndc-key" { algorithm hmac-sha256; secret "abc"; };
view "internal" {
	match-clients { any; };
	zone "corp.example" { type master; };
};
`)
	require.NoError(t, err)

	count := 0
	bind9EachZone(stmts, func(view string, z bind9.Statement) { count++ })
	assert.Equal(t, 1, count)
}
