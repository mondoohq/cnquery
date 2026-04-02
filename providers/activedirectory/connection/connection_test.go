// Copyright Mondoo, Inc. 2024, 2026
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

func TestResolveLDAPTransport(t *testing.T) {
	tests := []struct {
		name          string
		opts          map[string]string
		wantTransport ldapTransport
		wantPort      int
		wantTLS       bool
		wantStartTLS  bool
		wantErr       string
	}{
		{
			name:          "default transport is ldaps",
			opts:          map[string]string{},
			wantTransport: ldapTransportLDAPS,
			wantPort:      636,
			wantTLS:       true,
		},
		{
			name:          "explicit ldaps stays ldaps",
			opts:          map[string]string{OptionLDAPS: "true"},
			wantTransport: ldapTransportLDAPS,
			wantPort:      636,
			wantTLS:       true,
		},
		{
			name:          "starttls uses ldap port with TLS upgrade",
			opts:          map[string]string{OptionStartTLS: "true"},
			wantTransport: ldapTransportStartTLS,
			wantPort:      389,
			wantTLS:       true,
			wantStartTLS:  true,
		},
		{
			name:          "plain ldap is explicit",
			opts:          map[string]string{OptionPlainLDAP: "true"},
			wantTransport: ldapTransportPlain,
			wantPort:      389,
		},
		{
			name:    "conflicting transport options error",
			opts:    map[string]string{OptionLDAPS: "true", OptionPlainLDAP: "true"},
			wantErr: "LDAP transport options are mutually exclusive: choose one of ldaps, plain-ldap, or starttls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := resolveLDAPTransport(tt.opts)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q", tt.wantErr)
				}
				if got := err.Error(); got != tt.wantErr {
					t.Fatalf("error = %q, want %q", got, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if transport != tt.wantTransport {
				t.Fatalf("transport = %q, want %q", transport, tt.wantTransport)
			}
			if got := defaultPort(transport); got != tt.wantPort {
				t.Fatalf("defaultPort(%q) = %d, want %d", transport, got, tt.wantPort)
			}
			if got := transport.usesTLS(); got != tt.wantTLS {
				t.Fatalf("usesTLS(%q) = %v, want %v", transport, got, tt.wantTLS)
			}
			if got := transport.usesStartTLS(); got != tt.wantStartTLS {
				t.Fatalf("usesStartTLS(%q) = %v, want %v", transport, got, tt.wantStartTLS)
			}
		})
	}
}

func TestSelectKerberosAuthSource(t *testing.T) {
	tests := []struct {
		name       string
		user       string
		password   string
		opts       map[string]string
		wantSource kerberosAuthSource
		wantErr    string
	}{
		{
			name:    "keytab requires user",
			opts:    map[string]string{OptionKeytab: "/tmp/test.keytab"},
			wantErr: "--user is required with --keytab to identify the principal",
		},
		{
			name:       "keytab with user",
			user:       "alice@CORP.EXAMPLE.COM",
			opts:       map[string]string{OptionKeytab: "/tmp/test.keytab"},
			wantSource: kerberosAuthSourceKeytab,
		},
		{
			name:       "ccache without user is fine",
			opts:       map[string]string{OptionCCache: "/tmp/krb5cc"},
			wantSource: kerberosAuthSourceCCache,
		},
		{
			name:     "password requires user",
			password: "secret",
			opts:     map[string]string{},
			wantErr:  "--user is required with --password for Kerberos authentication",
		},
		{
			name:       "password with user",
			user:       "alice@CORP.EXAMPLE.COM",
			password:   "secret",
			opts:       map[string]string{},
			wantSource: kerberosAuthSourcePassword,
		},
		{
			name:       "implicit current session without explicit creds",
			opts:       map[string]string{},
			wantSource: kerberosAuthSourceCurrentSession,
		},
		{
			name:    "explicit user without kerberos material is rejected",
			user:    "alice@CORP.EXAMPLE.COM",
			opts:    map[string]string{},
			wantErr: "--kerberos with --user also requires --password, --keytab, or --ccache; omit --user to use the current Windows session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := selectKerberosAuthSource(tt.user, tt.password, tt.opts)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q", tt.wantErr)
				}
				if got := err.Error(); got != tt.wantErr {
					t.Fatalf("error = %q, want %q", got, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if source != tt.wantSource {
				t.Fatalf("source = %q, want %q", source, tt.wantSource)
			}
		})
	}
}
