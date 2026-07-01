// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAddress(t *testing.T) {
	cases := []struct {
		name       string
		address    string
		wantHost   string
		wantScheme string
		wantBase   string
		wantErr    bool
	}{
		{"bare host defaults to https and /api", "portainer.example.com", "portainer.example.com", "https", "/api", false},
		{"explicit http scheme is kept", "http://portainer.example.com", "portainer.example.com", "http", "/api", false},
		{"explicit port is preserved", "https://portainer.example.com:9443", "portainer.example.com:9443", "https", "/api", false},
		{"bare host:port defaults scheme and keeps port", "portainer.example.com:9443", "portainer.example.com:9443", "https", "/api", false},
		{"reverse-proxy prefix gets /api appended", "https://portainer.example.com/prefix", "portainer.example.com", "https", "/prefix/api", false},
		{"trailing slash keeps the default base path", "https://portainer.example.com/", "portainer.example.com", "https", "/api", false},
		{"surrounding whitespace is trimmed", "  https://portainer.example.com  ", "portainer.example.com", "https", "/api", false},
		{"bare host with whitespace still defaults scheme", "  portainer.example.com  ", "portainer.example.com", "https", "/api", false},
		{"empty address is an error", "", "", "", "", true},
		{"whitespace-only address is an error", "   ", "", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host, scheme, base, err := parseAddress(tc.address)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantHost, host, "host")
			assert.Equal(t, tc.wantScheme, scheme, "scheme")
			assert.Equal(t, tc.wantBase, base, "basePath")
		})
	}
}

func TestAuthenticationMethod(t *testing.T) {
	cases := map[int64]string{1: "internal", 2: "ldap", 3: "oauth", 0: "unknown", 99: "unknown"}
	for in, want := range cases {
		assert.Equalf(t, want, AuthenticationMethod(in), "AuthenticationMethod(%d)", in)
	}
}

func TestUserRole(t *testing.T) {
	cases := map[int64]string{1: "administrator", 2: "standard", 0: "unknown", 99: "unknown"}
	for in, want := range cases {
		assert.Equalf(t, want, UserRole(in), "UserRole(%d)", in)
	}
}

func TestMembershipRole(t *testing.T) {
	cases := map[int64]string{1: "leader", 2: "member", 0: "unknown", 99: "unknown"}
	for in, want := range cases {
		assert.Equalf(t, want, MembershipRole(in), "MembershipRole(%d)", in)
	}
}

func TestEnvironmentStatus(t *testing.T) {
	cases := map[int64]string{1: "up", 2: "down", 0: "unknown", 99: "unknown"}
	for in, want := range cases {
		assert.Equalf(t, want, EnvironmentStatus(in), "EnvironmentStatus(%d)", in)
	}
}
