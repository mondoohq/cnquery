// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSystemdShowOutput(t *testing.T) {
	input := `Description=Multi-User System
LoadState=loaded
ActiveState=active
SubState=active
UnitFileState=enabled
FragmentPath=/lib/systemd/system/multi-user.target
Wants=systemd-logind.service systemd-user-sessions.service
Requires=basic.target
After=basic.target rescue.service rescue.target
Environment=FOO=bar BAZ=qux
`
	props := parseSystemdShowOutput(input)
	assert.Equal(t, "Multi-User System", props["Description"])
	assert.Equal(t, "loaded", props["LoadState"])
	assert.Equal(t, "active", props["ActiveState"])
	assert.Equal(t, "enabled", props["UnitFileState"])
	assert.Equal(t, "/lib/systemd/system/multi-user.target", props["FragmentPath"])
	assert.Equal(t, "systemd-logind.service systemd-user-sessions.service", props["Wants"])
	assert.Equal(t, "basic.target", props["Requires"])

	// Value containing '=' must be preserved verbatim (split on first '=' only).
	assert.Equal(t, "FOO=bar BAZ=qux", props["Environment"])
}

func TestParseSystemdShowOutputEmptyValue(t *testing.T) {
	// Optional properties get reported as `Key=` (empty value).
	input := `LoadState=loaded
UnitFileState=
After=
`
	props := parseSystemdShowOutput(input)
	assert.Equal(t, "loaded", props["LoadState"])
	assert.Equal(t, "", props["UnitFileState"])
	_, ok := props["After"]
	assert.True(t, ok, "empty values should still be recorded")
}

func TestSplitSystemdUnitList(t *testing.T) {
	assert.Equal(t, []any{}, splitSystemdUnitList(""))
	assert.Equal(t, []any{"basic.target"}, splitSystemdUnitList("basic.target"))
	assert.Equal(t,
		[]any{"systemd-logind.service", "systemd-user-sessions.service"},
		splitSystemdUnitList("systemd-logind.service systemd-user-sessions.service"))
	// Extra whitespace collapses.
	assert.Equal(t,
		[]any{"a.service", "b.service"},
		splitSystemdUnitList("  a.service   b.service  "))
}

func TestShellQuoteUnit(t *testing.T) {
	// Normal unit names pass through unchanged.
	assert.Equal(t, "multi-user.target", shellQuoteUnit("multi-user.target"))
	assert.Equal(t, "getty@tty1.service", shellQuoteUnit("getty@tty1.service"))
	// Backslash is a shell metacharacter, so escaped unit names (rare but
	// possible in device units like `dev-disk-by\x2dlabel-root.device`) get
	// quoted defensively. The backslash survives literally inside '...'.
	assert.Equal(t, `'dev-disk-by\x2dlabel-root.device'`, shellQuoteUnit(`dev-disk-by\x2dlabel-root.device`))

	// Empty string -> ''.
	assert.Equal(t, "''", shellQuoteUnit(""))

	// Defensive quoting against unexpected characters.
	assert.Equal(t, "'has space'", shellQuoteUnit("has space"))
	assert.Equal(t, `'it'\''s'`, shellQuoteUnit("it's"))
}

func TestListSystemdTargetNames_Parse(t *testing.T) {
	// Verify the de-dup + suffix-stripping logic used by listSystemdTargetNames,
	// driving it through the same string-processing path it would see at runtime.
	installed := `basic.target                       static
default.target                     alias
multi-user.target                  enabled
graphical.target                   static
`
	active := `basic.target              loaded active active   Basic System
multi-user.target         loaded active active   Multi-User System
sysinit.target            loaded active active   System Initialization
`
	seen := map[string]struct{}{}
	var names []string
	for _, line := range strings.Split(installed+"\n"+active, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		unit := fields[0]
		if !strings.HasSuffix(unit, ".target") {
			continue
		}
		name := strings.TrimSuffix(unit, ".target")
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}

	// Order preserved as encountered; duplicates dropped; suffixes stripped.
	require.Equal(t,
		[]string{"basic", "default", "multi-user", "graphical", "sysinit"},
		names)
}

func TestSplitSystemctlShowBlocks(t *testing.T) {
	// Two unit blocks separated by a single blank line, like `systemctl
	// show --no-pager -- basic.target multi-user.target` emits.
	input := `Description=Basic System
LoadState=loaded
ActiveState=active

Description=Multi-User System
LoadState=loaded
ActiveState=active
`
	blocks := splitSystemctlShowBlocks(input)
	require.Len(t, blocks, 2)
	assert.Contains(t, blocks[0], "Description=Basic System")
	assert.Contains(t, blocks[1], "Description=Multi-User System")

	// Each block parses independently.
	first := parseSystemdShowOutput(blocks[0])
	second := parseSystemdShowOutput(blocks[1])
	assert.Equal(t, "Basic System", first["Description"])
	assert.Equal(t, "Multi-User System", second["Description"])

	// Trailing blank lines (common from systemctl) don't produce an empty
	// final block.
	withTrailing := input + "\n\n"
	assert.Len(t, splitSystemctlShowBlocks(withTrailing), 2)

	// CRLF survives normalization.
	crlf := strings.ReplaceAll(input, "\n", "\r\n")
	crlfBlocks := splitSystemctlShowBlocks(crlf)
	require.Len(t, crlfBlocks, 2)
	assert.Contains(t, crlfBlocks[0], "Description=Basic System")

	// Empty input returns no blocks.
	assert.Empty(t, splitSystemctlShowBlocks(""))
}
