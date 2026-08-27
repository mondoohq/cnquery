// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/providers/os/registry"
)

func dwordItem(key string, n int64) registry.RegistryKeyItem {
	return registry.RegistryKeyItem{Key: key, Value: registry.RegistryKeyValue{Kind: registry.DWORD, Number: n}}
}

func szItem(key, s string) registry.RegistryKeyItem {
	return registry.RegistryKeyItem{Key: key, Value: registry.RegistryKeyValue{Kind: registry.SZ, String: s}}
}

func TestRegistryItemInt(t *testing.T) {
	t.Run("DWORD value", func(t *testing.T) {
		n, ok := registryItemInt(dwordItem("MaxSize", 196608))
		assert.True(t, ok)
		assert.Equal(t, int64(196608), n)
	})

	t.Run("string decimal (Group Policy form)", func(t *testing.T) {
		n, ok := registryItemInt(szItem("Retention", "0"))
		assert.True(t, ok)
		assert.Equal(t, int64(0), n)
	})

	t.Run("string hex 0xFFFFFFFF", func(t *testing.T) {
		n, ok := registryItemInt(szItem("Retention", "0xFFFFFFFF"))
		assert.True(t, ok)
		assert.Equal(t, int64(0xFFFFFFFF), n)
	})

	t.Run("string -1", func(t *testing.T) {
		n, ok := registryItemInt(szItem("Retention", "-1"))
		assert.True(t, ok)
		assert.Equal(t, int64(-1), n)
	})

	t.Run("non-numeric string", func(t *testing.T) {
		_, ok := registryItemInt(szItem("Retention", "nope"))
		assert.False(t, ok)
	})
}

func TestDecodeRetention(t *testing.T) {
	assert.Equal(t, retentionOverwriteAsNeeded, decodeRetention(0))
	assert.Equal(t, retentionNeverOverwrite, decodeRetention(0xFFFFFFFF))
	assert.Equal(t, retentionNeverOverwrite, decodeRetention(-1))
	// a positive seconds value (legacy "overwrite by days")
	assert.Equal(t, retentionOverwriteByDays, decodeRetention(604800))
}

func TestLookupInt(t *testing.T) {
	items := map[string]registry.RegistryKeyItem{
		"maxsize":   dwordItem("MaxSize", 32768),
		"retention": szItem("Retention", "0"),
	}

	n, ok := lookupInt(items, "maxsize")
	assert.True(t, ok)
	assert.Equal(t, int64(32768), n)

	_, ok = lookupInt(items, "missing")
	assert.False(t, ok)
}

func TestResolveMaxSizeKB(t *testing.T) {
	dword := func(v int64) map[string]registry.RegistryKeyItem {
		return map[string]registry.RegistryKeyItem{"maxsize": dwordItem("MaxSize", v)}
	}
	empty := map[string]registry.RegistryKeyItem{}

	t.Run("Group Policy wins and is already in KB", func(t *testing.T) {
		n, ok := resolveMaxSizeKB(dword(196608), dword(20480*1024), dword(1024*1024))
		assert.True(t, ok)
		assert.Equal(t, int64(196608), n)
	})

	t.Run("the classic channel configuration is in bytes", func(t *testing.T) {
		n, ok := resolveMaxSizeKB(empty, dword(20480*1024), empty)
		assert.True(t, ok)
		assert.Equal(t, int64(20480), n)
	})

	// Measured on a live host: Microsoft-Windows-PowerShell/Operational
	// carries MaxSize 0xf00000 under WINEVT and nothing under
	// Services\EventLog, so without this source it reported the documented
	// 20480 default instead of its real 15360.
	t.Run("a modern channel resolves through WINEVT", func(t *testing.T) {
		n, ok := resolveMaxSizeKB(empty, empty, dword(15728640))
		assert.True(t, ok)
		assert.Equal(t, int64(15360), n)
	})

	t.Run("a classic log never reaches WINEVT", func(t *testing.T) {
		// Services\EventLog says 20 MB, WINEVT says 1 MB; the classic source wins
		n, ok := resolveMaxSizeKB(empty, dword(20480*1024), dword(1024*1024))
		assert.True(t, ok)
		assert.Equal(t, int64(20480), n)
	})

	// The registry holds overrides only. Measured on a live host:
	// Microsoft-Windows-WinRM/Operational has no MaxSize value anywhere and
	// runs at the 1052672 bytes its manifest declares, so a miss has to be
	// reported rather than resolved, or the caller never consults the one
	// source that knows the manifest value.
	t.Run("nothing configured is a miss, not a default", func(t *testing.T) {
		n, ok := resolveMaxSizeKB(empty, empty, empty)
		assert.False(t, ok)
		assert.Equal(t, int64(0), n)

		n, ok = resolveMaxSizeKB(nil, nil, nil)
		assert.False(t, ok)
		assert.Equal(t, int64(0), n)
	})
}

func TestDecodeChannelRetention(t *testing.T) {
	// The WINEVT channel key is a flag, not the seconds period the classic
	// Services\EventLog key uses. Feeding a 1 to decodeRetention would report
	// a channel configured to retain its events as one that overwrites them
	// after a single day, which is the mistake this decode exists to avoid.
	assert.Equal(t, retentionOverwriteAsNeeded, decodeChannelRetention(0))
	assert.Equal(t, retentionNeverOverwrite, decodeChannelRetention(1))
	assert.Equal(t, retentionNeverOverwrite, decodeChannelRetention(-1))

	assert.Equal(t, retentionOverwriteByDays, decodeRetention(1),
		"the seconds encoding must keep reading a 1 as a period, so the two decodes stay distinct")
}

func TestDecodeLogMode(t *testing.T) {
	tests := []struct {
		mode string
		want string
		ok   bool
	}{
		{mode: "Circular", want: retentionOverwriteAsNeeded, ok: true},
		{mode: "circular", want: retentionOverwriteAsNeeded, ok: true},
		// Both of these keep the events: Retain refuses new ones once full,
		// AutoBackup archives the log and starts a new one. Neither overwrites.
		{mode: "Retain", want: retentionNeverOverwrite, ok: true},
		{mode: "AutoBackup", want: retentionNeverOverwrite, ok: true},
		{mode: " Retain ", want: retentionNeverOverwrite, ok: true},
		// An unrecognized or absent mode must not resolve to anything, so the
		// caller falls through to the documented default rather than guessing.
		{mode: "", ok: false},
		{mode: "Whatever", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got, ok := decodeLogMode(tt.mode)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
