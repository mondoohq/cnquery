// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGlobalState(t *testing.T) {
	for name, tt := range map[string]struct {
		stdout string
		want   int64
		ok     bool
	}{
		"enabled":     {"Firewall is enabled. (State = 1)\n", 1, true},
		"block all":   {"Firewall is enabled. (State = 2)\n", 2, true},
		"disabled":    {"Firewall is disabled. (State = 0)\n", 0, true},
		"no state no": {"Firewall is enabled.\n", 1, true},
		"managed":     {"Firewall settings cannot be modified from command line on managed Mac computers.\n", 0, false},
		"empty":       {"", 0, false},
	} {
		t.Run(name, func(t *testing.T) {
			v, ok := parseGlobalState(tt.stdout)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, v)
			}
		})
	}
}

// An unreadable setting must not collapse to false. "the firewall is off" and
// "we could not read the firewall" are different findings.
func TestParseOnOff(t *testing.T) {
	for name, tt := range map[string]struct {
		stdout string
		want   bool
		ok     bool
	}{
		"stealth on":     {"Firewall stealth mode is on\n", true, true},
		"stealth off":    {"Firewall stealth mode is off\n", false, true},
		"blockall on":    {"Firewall has block all state set to enabled.\n", true, true},
		"blockall off":   {"Firewall has block all state set to disabled.\n", false, true},
		"managed mac":    {"Firewall settings cannot be modified from command line on managed Mac computers.\n", false, false},
		"unrecognised":   {"something else entirely\n", false, false},
		"empty is not a": {"", false, false},
		// A bare "enabled."/"disabled." is not a toggle answer. The
		// --getallowsigned reply ends that way, and a logging line can too, so
		// matching the suffix would let one getter be read as another.
		"logging detail line": {"Logging enabled. Detail level: brief\n", false, false},
		"allowsigned line":    {"Automatically allow built-in signed software ENABLED.\n", false, false},
		// "is on" must be a whole word, not a prefix of something else.
		"is one": {"Firewall stealth mode is one of several settings\n", false, false},
	} {
		t.Run(name, func(t *testing.T) {
			v, ok := parseOnOff(tt.stdout)
			assert.Equal(t, tt.ok, ok, "recognised")
			if tt.ok {
				assert.Equal(t, tt.want, v)
			}
		})
	}
}

func TestParseAllowSigned(t *testing.T) {
	builtin, downloaded, ok := parseAllowSigned(
		"Automatically allow built-in signed software ENABLED. \n" +
			"Automatically allow downloaded signed software ENABLED. \n")
	assert.True(t, ok)
	assert.True(t, builtin)
	assert.True(t, downloaded)

	builtin, downloaded, ok = parseAllowSigned(
		"Automatically allow built-in signed software ENABLED. \n" +
			"Automatically allow downloaded signed software DISABLED. \n")
	assert.True(t, ok)
	assert.True(t, builtin)
	assert.False(t, downloaded)

	// A partial reply must not be reported as two falses.
	_, _, ok = parseAllowSigned("Automatically allow built-in signed software ENABLED. \n")
	assert.False(t, ok)

	_, _, ok = parseAllowSigned("Firewall settings cannot be modified from command line on managed Mac computers.\n")
	assert.False(t, ok)
}
