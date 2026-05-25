// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// kernel.cmdline parser
// =============================================================================

func TestParseKernelCmdline_Typical(t *testing.T) {
	// Typical Debian/Ubuntu boot line.
	raw := `BOOT_IMAGE=/boot/vmlinuz-6.5.0-1-amd64 root=UUID=abcd-1234 ro quiet splash mitigations=auto`
	params, flags := parseKernelCmdline(raw)

	assert.Equal(t, "/boot/vmlinuz-6.5.0-1-amd64", params["BOOT_IMAGE"])
	assert.Equal(t, "UUID=abcd-1234", params["root"], "value containing '=' is preserved verbatim")
	assert.Equal(t, "auto", params["mitigations"])

	// `ro`, `quiet`, `splash` are bare flags.
	assert.Contains(t, flags, "ro")
	assert.Contains(t, flags, "quiet")
	assert.Contains(t, flags, "splash")
	assert.NotContains(t, flags, "mitigations")
}

func TestParseKernelCmdline_DuplicateLastWins(t *testing.T) {
	// `console=` typically appears twice (serial + virtual terminal).
	raw := `console=tty0 console=ttyS0,115200n8`
	params, flags := parseKernelCmdline(raw)
	assert.Equal(t, "ttyS0,115200n8", params["console"], "last occurrence wins")
	assert.Empty(t, flags)
}

func TestParseKernelCmdline_Empty(t *testing.T) {
	params, flags := parseKernelCmdline("")
	assert.Empty(t, params)
	assert.Empty(t, flags)
}

func TestParseKernelCmdline_EmptyValue(t *testing.T) {
	// `key=` is a parameter with an empty string value, not a flag.
	raw := `root=UUID=x foo= quiet`
	params, flags := parseKernelCmdline(raw)
	assert.Equal(t, "", params["foo"])
	_, ok := params["foo"]
	assert.True(t, ok, "key= must produce a parameter, not be dropped")
	assert.Contains(t, flags, "quiet")
	assert.NotContains(t, flags, "foo")
}

func TestParseKernelCmdline_LeadingEquals(t *testing.T) {
	// A token starting with `=` has no key; treat it as a flag so we
	// don't index a map by an empty string.
	raw := `=weird ro`
	params, flags := parseKernelCmdline(raw)
	assert.Contains(t, flags, "=weird")
	assert.Contains(t, flags, "ro")
	_, ok := params[""]
	assert.False(t, ok, "empty-key parameters must not appear in the map")
}

// =============================================================================
// kernel.taint
// =============================================================================

func TestTaintReasons_Clean(t *testing.T) {
	assert.Empty(t, taintReasons(0))
}

func TestTaintReasons_SingleBit(t *testing.T) {
	// Bit 12 = O = out-of-tree module loaded (the common case on
	// systems running e.g. VirtualBox kernel modules).
	r := taintReasons(1 << 12)
	require.Len(t, r, 1)
	assert.Equal(t, "out-of-tree module loaded", r[0])
}

func TestTaintReasons_MultipleBits(t *testing.T) {
	// Bits 0 (proprietary), 7 (oops), 12 (out-of-tree).
	bitmask := int64((1 << 0) | (1 << 7) | (1 << 12))
	r := taintReasons(bitmask)
	require.Len(t, r, 3)
	assert.Equal(t, "proprietary module loaded", r[0])
	assert.Equal(t, "kernel oops or BUG", r[1])
	assert.Equal(t, "out-of-tree module loaded", r[2])
}

func TestTaintReasons_UnknownBitsIgnored(t *testing.T) {
	// Set a high bit beyond what we have a name for. The known bits
	// should still resolve; the unknown bit is silently skipped rather
	// than poisoning the list.
	bitmask := int64((1 << 0) | (1 << 30))
	r := taintReasons(bitmask)
	require.Len(t, r, 1)
	assert.Equal(t, "proprietary module loaded", r[0])
}

// =============================================================================
// kernel.lockdown
// =============================================================================

func TestParseLockdownMode_None(t *testing.T) {
	// Default state: lockdown LSM compiled in but not engaged.
	assert.Equal(t, "none", parseLockdownMode("[none] integrity confidentiality\n"))
}

func TestParseLockdownMode_Integrity(t *testing.T) {
	assert.Equal(t, "integrity", parseLockdownMode("none [integrity] confidentiality\n"))
}

func TestParseLockdownMode_Confidentiality(t *testing.T) {
	assert.Equal(t, "confidentiality", parseLockdownMode("none integrity [confidentiality]\n"))
}

func TestParseLockdownMode_Empty(t *testing.T) {
	// Empty content → kernel returned the file but with nothing in it.
	// We don't claim "none" here because that would be wrong.
	assert.Equal(t, "unknown", parseLockdownMode(""))
	assert.Equal(t, "unknown", parseLockdownMode("\n"))
}

func TestParseLockdownMode_NoBracketedToken(t *testing.T) {
	// Malformed: no token has brackets. Don't guess.
	assert.Equal(t, "unknown", parseLockdownMode("none integrity confidentiality"))
}

// =============================================================================
// kernel.aslr
// =============================================================================

func TestAslrLevel(t *testing.T) {
	assert.Equal(t, "disabled", aslrLevel(0))
	assert.Equal(t, "conservative", aslrLevel(1))
	assert.Equal(t, "full", aslrLevel(2))
	// CIS benchmarks require 2; anything outside the documented set is
	// "unknown" rather than guessed.
	assert.Equal(t, "unknown", aslrLevel(3))
	assert.Equal(t, "unknown", aslrLevel(-1))
}

// =============================================================================
// helpers
// =============================================================================

func TestStringsAsAnySlice(t *testing.T) {
	out := stringsAsAnySlice([]string{"a", "b"})
	require.Len(t, out, 2)
	assert.Equal(t, "a", out[0])
	assert.Equal(t, "b", out[1])
	assert.Empty(t, stringsAsAnySlice(nil))
}
