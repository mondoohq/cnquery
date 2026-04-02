// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"reflect"
	"testing"
)

func TestParseGroupType(t *testing.T) {
	tests := []struct {
		name      string
		raw       int64
		wantLabel string
		wantRaw   int64
	}{
		// Security groups (bit 31 set — stored as negative int32 values in AD).
		{"security global", -2147483646, "Security - Global", -2147483646},            // 0x80000002
		{"security domain-local", -2147483644, "Security - DomainLocal", -2147483644}, // 0x80000004
		{"security universal", -2147483640, "Security - Universal", -2147483640},      // 0x80000008

		// Distribution groups (bit 31 clear — positive values).
		{"distribution global", 2, "Distribution - Global", 2},
		{"distribution domain-local", 4, "Distribution - DomainLocal", 4},
		{"distribution universal", 8, "Distribution - Universal", 8},

		// Edge cases.
		{"no scope bits", 0, "Distribution - Unknown", 0},
		{"security no scope", -2147483648, "Security - Unknown", -2147483648}, // 0x80000000 only
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, raw := parseGroupType(tt.raw)
			if label != tt.wantLabel {
				t.Errorf("parseGroupType(%d) label = %q, want %q", tt.raw, label, tt.wantLabel)
			}
			if raw != tt.wantRaw {
				t.Errorf("parseGroupType(%d) raw = %d, want %d", tt.raw, raw, tt.wantRaw)
			}
		})
	}
}

func TestParseInt64Attr(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int64
	}{
		{"positive number", "12345", 12345},
		{"negative number", "-864000000000", -864000000000},
		{"empty string", "", 0},
		{"whitespace padded", "  42  ", 42},
		{"parse error", "abc", 0},
		{"zero", "0", 0},
		{"null-byte terminated", "100\x00", 100},
		{"null-byte padded", "200\x00\x00", 200},
		{"large positive", "9223372036854775807", 9223372036854775807},   // max int64
		{"large negative", "-9223372036854775808", -9223372036854775808}, // min int64
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInt64Attr(tt.s)
			if got != tt.want {
				t.Errorf("parseInt64Attr(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestBestMatchingSearchBase(t *testing.T) {
	searchBases := []string{
		"DC=mini,DC=lab",
		"DC=child,DC=mini,DC=lab",
	}

	tests := []struct {
		name     string
		memberDN string
		want     string
	}{
		{
			name:     "matches root domain base",
			memberDN: "CN=Alice,CN=Users,DC=mini,DC=lab",
			want:     "DC=mini,DC=lab",
		},
		{
			name:     "prefers longest matching child base",
			memberDN: "CN=Bob,CN=Users,DC=child,DC=mini,DC=lab",
			want:     "DC=child,DC=mini,DC=lab",
		},
		{
			name:     "returns empty when no base matches",
			memberDN: "CN=Carol,CN=Users,DC=other,DC=lab",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bestMatchingSearchBase(tt.memberDN, searchBases)
			if got != tt.want {
				t.Fatalf("bestMatchingSearchBase(%q) = %q, want %q", tt.memberDN, got, tt.want)
			}
		})
	}
}

func TestMemberLookupBatches(t *testing.T) {
	searchBases := []string{
		"DC=mini,DC=lab",
		"DC=child,DC=mini,DC=lab",
	}
	memberDNs := []string{
		"CN=ChildOne,CN=Users,DC=child,DC=mini,DC=lab",
		"CN=BaseOne,CN=Users,DC=mini,DC=lab",
		"CN=ChildTwo,CN=Users,DC=child,DC=mini,DC=lab",
		"CN=BaseTwo,CN=Users,DC=mini,DC=lab",
		"CN=BaseThree,CN=Users,DC=mini,DC=lab",
	}

	got := memberLookupBatches(memberDNs, searchBases, 2)
	want := []groupMemberLookupBatch{
		{
			searchBase: "DC=mini,DC=lab",
			memberDNs: []string{
				"CN=BaseOne,CN=Users,DC=mini,DC=lab",
				"CN=BaseTwo,CN=Users,DC=mini,DC=lab",
			},
		},
		{
			searchBase: "DC=mini,DC=lab",
			memberDNs: []string{
				"CN=BaseThree,CN=Users,DC=mini,DC=lab",
			},
		},
		{
			searchBase: "DC=child,DC=mini,DC=lab",
			memberDNs: []string{
				"CN=ChildOne,CN=Users,DC=child,DC=mini,DC=lab",
				"CN=ChildTwo,CN=Users,DC=child,DC=mini,DC=lab",
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("memberLookupBatches() = %#v, want %#v", got, want)
	}
}
