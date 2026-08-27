// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers/os/resources/bind9"
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

// A key statement is legal inside a view, and two views may declare different
// keys under the same name. The view is therefore part of what identifies a
// key: an id built from the name alone makes the second key resolve to the
// cached first one, which reports the wrong algorithm — and reports it as the
// stronger of the two when the weaker one is in the internet-facing view.
func TestBind9KeyIdentityIncludesTheView(t *testing.T) {
	stmts, err := bind9.Parse(`
key "top-level" { algorithm hmac-sha512; secret "a"; };
view "internal" {
	key "shared-name" { algorithm hmac-sha256; secret "b"; };
};
view "external" {
	key "shared-name" { algorithm hmac-md5; secret "c"; };
};
`)
	require.NoError(t, err)

	// the ids the resource builds, in the order it builds them
	var ids []string
	var algorithms []string
	collect := func(block []bind9.Statement, view string) {
		for _, k := range bind9.Find(block, "key") {
			ids = append(ids, "/etc/bind/named.conf/key/"+view+"/"+k.Arg(0))
			algorithms = append(algorithms, bind9.Value(k.Block, "algorithm"))
		}
	}
	collect(stmts, "")
	for _, v := range bind9.Find(stmts, "view") {
		collect(v.Block, v.Arg(0))
	}

	assert.Equal(t, []string{
		"/etc/bind/named.conf/key//top-level",
		"/etc/bind/named.conf/key/internal/shared-name",
		"/etc/bind/named.conf/key/external/shared-name",
	}, ids)
	assert.Equal(t, []string{"hmac-sha512", "hmac-sha256", "hmac-md5"}, algorithms)

	seen := map[string]bool{}
	for _, id := range ids {
		require.False(t, seen[id], "two keys share the id %q, so one shadows the other in the resource cache", id)
		seen[id] = true
	}
}
