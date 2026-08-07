// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"testing"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func newTestConn(t *testing.T, opts map[string]string) *ElasticsearchConnection {
	t.Helper()
	if opts == nil {
		opts = map[string]string{}
	}
	if opts[OptionHost] == "" {
		opts[OptionHost] = "es.example.com"
	}
	conf := &inventory.Config{Type: "elasticsearch", Options: opts}
	asset := &inventory.Asset{Connections: []*inventory.Config{conf}}
	conn, err := NewElasticsearchConnection(1, asset, conf)
	if err != nil {
		t.Fatalf("NewElasticsearchConnection: %v", err)
	}
	return conn
}

func TestAddressDefaults(t *testing.T) {
	// Default scheme https and port 9200.
	if got := newTestConn(t, nil).address(); got != "https://es.example.com:9200" {
		t.Errorf("address = %q", got)
	}
	c := newTestConn(t, map[string]string{OptionScheme: "http", OptionPort: "9201"})
	if got := c.address(); got != "http://es.example.com:9201" {
		t.Errorf("address = %q", got)
	}
}

func TestTransportDisabledForPlainHTTP(t *testing.T) {
	// http with no TLS material needs no custom transport.
	c := newTestConn(t, map[string]string{OptionScheme: "http"})
	tr, err := c.transport()
	if err != nil {
		t.Fatal(err)
	}
	if tr != nil {
		t.Error("plain http should not build a custom transport")
	}
}

func TestTransportInsecure(t *testing.T) {
	c := newTestConn(t, map[string]string{OptionScheme: "https", OptionTLSInsecure: "true"})
	tr, err := c.transport()
	if err != nil {
		t.Fatal(err)
	}
	if tr == nil || tr.TLSClientConfig == nil || !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("expected an insecure TLS transport")
	}
}

func TestTransportBadCAErrors(t *testing.T) {
	c := newTestConn(t, map[string]string{OptionScheme: "https", OptionTLSCA: "/no/such/ca.pem"})
	if _, err := c.transport(); err == nil {
		t.Error("expected an error for a missing CA file")
	}
}

func TestPermissionError(t *testing.T) {
	err := &PermissionError{Path: "/_security/user", StatusCode: 403}
	if !IsPermissionError(err) {
		t.Error("PermissionError should be classified as a permission error")
	}
	if IsPermissionError(errors.New("boom")) {
		t.Error("a plain error is not a permission error")
	}
	if IsPermissionError(nil) {
		t.Error("nil is not a permission error")
	}
}
