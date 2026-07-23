// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSPF(t *testing.T) {
	cases := []struct {
		name             string
		txt              string
		wantVersion      string
		wantMechanisms   []string
		wantAllQualifier string
	}{
		{
			name:             "softfail all",
			txt:              "v=spf1 include:_spf.google.com ip4:192.0.2.0/24 ~all",
			wantVersion:      "spf1",
			wantMechanisms:   []string{"include:_spf.google.com", "ip4:192.0.2.0/24", "~all"},
			wantAllQualifier: "~",
		},
		{
			name:             "hard fail all",
			txt:              "v=spf1 a mx -all",
			wantVersion:      "spf1",
			wantMechanisms:   []string{"a", "mx", "-all"},
			wantAllQualifier: "-",
		},
		{
			name:             "bare all defaults to pass qualifier",
			txt:              "v=spf1 a all",
			wantVersion:      "spf1",
			wantMechanisms:   []string{"a", "all"},
			wantAllQualifier: "+",
		},
		{
			name:             "no all mechanism",
			txt:              "v=spf1 include:_spf.example.com",
			wantVersion:      "spf1",
			wantMechanisms:   []string{"include:_spf.example.com"},
			wantAllQualifier: "",
		},
		{
			name:             "all substring in include is not the all mechanism",
			txt:              "v=spf1 include:spf.recall.example -all",
			wantVersion:      "spf1",
			wantMechanisms:   []string{"include:spf.recall.example", "-all"},
			wantAllQualifier: "-",
		},
		{
			name:             "first all wins when multiple are present",
			txt:              "v=spf1 ~all -all",
			wantVersion:      "spf1",
			wantMechanisms:   []string{"~all", "-all"},
			wantAllQualifier: "~",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			version, mechanisms, allQualifier := parseSPF(tc.txt)
			assert.Equal(t, tc.wantVersion, version)
			assert.Equal(t, tc.wantMechanisms, mechanisms)
			assert.Equal(t, tc.wantAllQualifier, allQualifier)
		})
	}
}

func TestParseDMARC(t *testing.T) {
	tags := parseDMARC("v=DMARC1; p=reject; sp=quarantine; rua=mailto:agg@example.com; ruf=mailto:f1@example.com, mailto:f2@example.com; pct=50; aspf=s; adkim=r")
	assert.Equal(t, "DMARC1", tags["v"])
	assert.Equal(t, "reject", tags["p"])
	assert.Equal(t, "quarantine", tags["sp"])
	assert.Equal(t, "s", tags["aspf"])
	assert.Equal(t, "r", tags["adkim"])
	assert.Equal(t, "50", tags["pct"])

	// case-insensitive keys, whitespace tolerant
	assert.Equal(t, "DMARC1", parseDMARC(" V=DMARC1 ; P=none ")["v"])
	assert.Equal(t, "none", parseDMARC(" V=DMARC1 ; P=none ")["p"])
}

func TestDmarcUris(t *testing.T) {
	assert.Equal(t, []any{"mailto:a@x.com", "mailto:b@y.com"}, dmarcUris("mailto:a@x.com, mailto:b@y.com"))
	assert.Equal(t, []any{}, dmarcUris(""))
	assert.Equal(t, []any{"mailto:a@x.com"}, dmarcUris("mailto:a@x.com"))
}

// TestDictTTL guards the JsonToDict key/type contract for a record's TTL: the
// map comes from json.Marshal/Unmarshal of dnsshake.DnsRecord, so the key is the
// json tag "ttl" (not "TTL") and the number is float64 (not int64). Reading the
// wrong key silently dropped the TTL; asserting .(int64) would panic.
func TestDictTTL(t *testing.T) {
	t.Run("reads the lowercase json-tag key as float64", func(t *testing.T) {
		got, ok := dictTTL(map[string]any{"ttl": float64(3600)})
		assert.True(t, ok)
		assert.Equal(t, int64(3600), got)
	})
	t.Run("uppercase TTL key is absent (the original bug)", func(t *testing.T) {
		_, ok := dictTTL(map[string]any{"TTL": float64(3600)})
		assert.False(t, ok)
	})
	t.Run("missing key", func(t *testing.T) {
		_, ok := dictTTL(map[string]any{"name": "example.com."})
		assert.False(t, ok)
	})
	t.Run("does not panic on an int64 value (wrong type)", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_, ok := dictTTL(map[string]any{"ttl": int64(3600)})
			assert.False(t, ok)
		})
	})
}
