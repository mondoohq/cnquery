// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"testing"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

func newTestConn(t *testing.T, opts map[string]string) *ClickhousedbConnection {
	t.Helper()
	if opts == nil {
		opts = map[string]string{}
	}
	if opts[OptionHost] == "" {
		opts[OptionHost] = "ch.example.com"
	}
	conf := &inventory.Config{
		Type:        "clickhousedb",
		Options:     opts,
		Credentials: []*vault.Credential{vault.NewPasswordCredential("", "")},
	}
	asset := &inventory.Asset{Connections: []*inventory.Config{conf}}
	conn, err := NewClickhousedbConnection(1, asset, conf)
	if err != nil {
		t.Fatalf("NewClickhousedbConnection: %v", err)
	}
	return conn
}

func TestServerIDAndDefaults(t *testing.T) {
	c := newTestConn(t, nil)
	if got := c.ServerID(); got != "ch.example.com:9000" {
		t.Errorf("ServerID = %q", got)
	}
	if c.database != "default" {
		t.Errorf("default database = %q, want default", c.database)
	}
	if c.user != "default" {
		t.Errorf("default user = %q, want default", c.user)
	}
	p := newTestConn(t, map[string]string{OptionPort: "9440"})
	if got := p.ServerID(); got != "ch.example.com:9440" {
		t.Errorf("ServerID (port) = %q", got)
	}
}

func TestMissingHostErrors(t *testing.T) {
	conf := &inventory.Config{Type: "clickhousedb", Options: map[string]string{}}
	asset := &inventory.Asset{Connections: []*inventory.Config{conf}}
	if _, err := NewClickhousedbConnection(1, asset, conf); err == nil {
		t.Error("expected an error when host is not set")
	}
}

func TestIsPermissionError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("code: 497, message: ... ACCESS_DENIED: ..."), true},
		{errors.New("Not enough privileges to read system.users"), true},
		{errors.New("code: 210, message: Connection refused"), false},
		{errors.New("network down"), false},
		{nil, false},
	}
	for _, c := range cases {
		if got := IsPermissionError(c.err); got != c.want {
			t.Errorf("IsPermissionError(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}
