// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"reflect"
	"testing"

	"github.com/miekg/dns"
)

func TestAddressesFromParams(t *testing.T) {
	testCases := []struct {
		name    string
		params  any
		want    []string
		wantErr bool
	}{
		{
			name: "single A record",
			params: map[string]any{
				"A": map[string]any{"rData": []any{"1.2.3.4"}},
			},
			want: []string{"1.2.3.4"},
		},
		{
			name: "multiple A records and AAAA",
			params: map[string]any{
				"A":    map[string]any{"rData": []any{"1.2.3.4", "5.6.7.8"}},
				"AAAA": map[string]any{"rData": []any{"2001:db8::1"}},
			},
			want: []string{"1.2.3.4", "5.6.7.8", "2001:db8::1"},
		},
		{
			name: "no address records",
			params: map[string]any{
				"MX": map[string]any{"rData": []any{"mail.example.com"}},
			},
			want: []string{},
		},
		{
			name: "empty and non-string rdata skipped",
			params: map[string]any{
				"A": map[string]any{"rData": []any{"1.2.3.4", "", 42}},
			},
			want: []string{"1.2.3.4"},
		},
		{
			name:    "wrong params type",
			params:  "not a map",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := addressesFromParams(tc.params)
			if (err != nil) != tc.wantErr {
				t.Fatalf("addressesFromParams() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("addressesFromParams() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestReverseAddrName documents the in-addr.arpa / ip6.arpa names the reverse
// field relies on — the transform that previously had to be hand-rolled in MQL.
func TestReverseAddrName(t *testing.T) {
	testCases := []struct {
		addr    string
		want    string
		wantErr bool
	}{
		{addr: "1.2.3.4", want: "4.3.2.1.in-addr.arpa."},
		{addr: "2001:db8::1", want: "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa."},
		{addr: "not-an-ip", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.addr, func(t *testing.T) {
			got, err := dns.ReverseAddr(tc.addr)
			if (err != nil) != tc.wantErr {
				t.Fatalf("dns.ReverseAddr(%q) error = %v, wantErr %v", tc.addr, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if got != tc.want {
				t.Errorf("dns.ReverseAddr(%q) = %q, want %q", tc.addr, got, tc.want)
			}
		})
	}
}
