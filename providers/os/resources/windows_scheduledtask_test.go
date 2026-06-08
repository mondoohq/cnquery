// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTriggerType(t *testing.T) {
	cases := map[string]string{
		"MSFT_TaskDailyTrigger":              "daily",
		"MSFT_TaskWeeklyTrigger":             "weekly",
		"MSFT_TaskMonthlyTrigger":            "monthly",
		"MSFT_TaskMonthlyDOWTrigger":         "monthlyDOW",
		"MSFT_TaskBootTrigger":               "boot",
		"MSFT_TaskLogonTrigger":              "logon",
		"MSFT_TaskRegistrationTrigger":       "registration",
		"MSFT_TaskTimeTrigger":               "time",
		"MSFT_TaskEventTrigger":              "event",
		"MSFT_TaskIdleTrigger":               "idle",
		"MSFT_TaskSessionStateChangeTrigger": "sessionStateChange",
		"":                                   "",
	}
	for in, want := range cases {
		assert.Equal(t, want, normalizeTriggerType(in), in)
	}
}

func TestParseWindowsTaskTime(t *testing.T) {
	// empty and never-ran sentinel both yield nil
	assert.Nil(t, parseWindowsTaskTime(""))
	assert.Nil(t, parseWindowsTaskTime("   "))
	assert.Nil(t, parseWindowsTaskTime("1899-11-30T00:00:00"))

	// CIM datetime string without timezone
	tm := parseWindowsTaskTime("2024-01-01T03:00:00")
	require.NotNil(t, tm)
	assert.Equal(t, 2024, tm.Year())
	assert.Equal(t, 3, tm.Hour())

	// round-trip ("o") formatted DateTime with offset
	tm = parseWindowsTaskTime("2024-03-10T07:00:00.0000000-07:00")
	require.NotNil(t, tm)
	assert.Equal(t, 2024, tm.Year())

	// legacy /Date(ms)/ form
	tm = parseWindowsTaskTime("/Date(1709044800000)/")
	require.NotNil(t, tm)
	assert.Equal(t, 2024, tm.Year())

	// unparseable input yields nil
	assert.Nil(t, parseWindowsTaskTime("not-a-date"))
}
