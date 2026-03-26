// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"crypto/tls"
	"testing"
)

func TestDomainToDN(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{"mini.lab", "DC=mini,DC=lab"},
		{"corp.example.com", "DC=corp,DC=example,DC=com"},
		{"single", "DC=single"},
		{"a.b.c.d.e", "DC=a,DC=b,DC=c,DC=d,DC=e"},
	}
	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			if got := domainToDN(tt.domain); got != tt.want {
				t.Errorf("domainToDN(%q) = %q, want %q", tt.domain, got, tt.want)
			}
		})
	}
}

func TestSplitPrincipal(t *testing.T) {
	tests := []struct {
		input     string
		wantUser  string
		wantRealm string
	}{
		{"alice@MINI.LAB", "alice", "MINI.LAB"},
		{"admin@CORP.EXAMPLE.COM", "admin", "CORP.EXAMPLE.COM"},
		{"alice", "alice", ""},
		{"", "", ""},
		{"user@realm@extra", "user@realm", "extra"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			user, realm := splitPrincipal(tt.input)
			if user != tt.wantUser || realm != tt.wantRealm {
				t.Errorf("splitPrincipal(%q) = (%q, %q), want (%q, %q)", tt.input, user, realm, tt.wantUser, tt.wantRealm)
			}
		})
	}
}

func TestResolveKrb5Conf(t *testing.T) {
	// Explicit path always wins.
	if got := resolveKrb5Conf("/custom/krb5.conf"); got != "/custom/krb5.conf" {
		t.Errorf("explicit: got %q, want /custom/krb5.conf", got)
	}

	// Empty explicit falls through to env or default.
	got := resolveKrb5Conf("")
	if got == "" {
		t.Error("resolveKrb5Conf('') returned empty string")
	}
}

func TestNewLDAPTLSConfig(t *testing.T) {
	tests := []struct {
		name     string
		server   string
		insecure bool
	}{
		{name: "strict verification", server: "dc01.example.com", insecure: false},
		{name: "insecure verification", server: "dc01.example.com", insecure: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newLDAPTLSConfig(tt.server, tt.insecure)
			if cfg.MinVersion != tls.VersionTLS12 {
				t.Fatalf("MinVersion = %v, want %v", cfg.MinVersion, tls.VersionTLS12)
			}
			if cfg.ServerName != tt.server {
				t.Fatalf("ServerName = %q, want %q", cfg.ServerName, tt.server)
			}
			if cfg.InsecureSkipVerify != tt.insecure {
				t.Fatalf("InsecureSkipVerify = %v, want %v", cfg.InsecureSkipVerify, tt.insecure)
			}
		})
	}
}
