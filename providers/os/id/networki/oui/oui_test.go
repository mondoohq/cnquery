// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package oui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/id/networki/oui"
)

func TestVendor(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{"bare oui", "00000c", "Cisco Systems"},
		{"bare oui, uppercase", "00000C", "Cisco Systems"},
		{"colon separated", "00:00:0c", "Cisco Systems"},
		{"colon separated, uppercase", "00:00:0C", "Cisco Systems"},
		{"full mac", "00:00:0c:11:22:33", "Cisco Systems"},
		{"full mac, no separators", "00000c112233", "Cisco Systems"},
		{"first record in the table", "000000", "XEROX"},
		{"vendor with a stripped suffix", "286fb9", "Nokia Shanghai Bell"},
		{"vendor kept whole", "08ea44", "Extreme Networks Headquarters"},
		{"apple", "3c22fb", "Apple"},

		// Separators are dropped wherever they fall, so a MAC written with
		// single-digit groups still resolves off its first six hex digits.
		{"ragged separators", "0:0:0:c:11:22", "NIPPON DEMPA"},

		{"unassigned", "ffffff", ""},
		{"empty", "", ""},
		{"too short", "00000", ""},
		{"too short after separators", "00:00:0", ""},
		{"not hex", "zzzzzz", ""},
		{"partially not hex", "00000z", ""},
		{"dash separated is not hex", "00-00-0c", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, oui.Vendor(test.addr))
		})
	}
}

// The point of the embedded table is that it costs read-only pages instead of
// heap, so a lookup must not allocate. A regression here means the table is
// being copied or decoded on the way out.
func TestVendorDoesNotAllocate(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_ = oui.Vendor("00:00:0c:11:22:33")
		_ = oui.Vendor("ffffff")
	})
	assert.Zero(t, allocs, "Vendor allocated")
}

// Binary search is only correct on a sorted table, and a truncated or
// misencoded blob would otherwise surface as silently missing vendors rather
// than as a failure.
func TestTableIsWellFormed(t *testing.T) {
	var assigned, prev int64 = 0, -1

	for v := int64(0); v < 1<<24; v++ {
		vendor := oui.Vendor(hex6(v))
		if vendor == "" {
			continue
		}
		require.Greater(t, v, prev, "records out of order")
		prev = v
		assigned++
	}

	// The IEEE MA-L registry holds tens of thousands of assignments. An exact
	// count would fail on every table refresh, but an empty or tiny table
	// means the blob did not load.
	assert.Greater(t, assigned, int64(30000), "table looks truncated")
}

func hex6(v int64) string {
	const digits = "0123456789abcdef"
	return string([]byte{
		digits[(v>>20)&0xf], digits[(v>>16)&0xf],
		digits[(v>>12)&0xf], digits[(v>>8)&0xf],
		digits[(v>>4)&0xf], digits[v&0xf],
	})
}

func BenchmarkVendor(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = oui.Vendor("00:00:0c:11:22:33")
	}
}
