// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWindowsUpdateHistory(t *testing.T) {
	r, err := os.Open("./testdata/update_history.json")
	require.NoError(t, err)
	defer r.Close()

	entries, err := ParseWindowsUpdateHistory(r)
	require.NoError(t, err)
	assert.Len(t, entries, 6)
	assert.Equal(t, "https://support.microsoft.com/help/5034441", entries[0].SupportUrl)
	assert.Equal(t, 1, entries[0].Operation)
	assert.Equal(t, 2, entries[0].ResultCode)
}

func TestParseWindowsUpdateHistory_Empty(t *testing.T) {
	entries, err := ParseWindowsUpdateHistory(strings.NewReader(""))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestFilterInstalledHistory(t *testing.T) {
	r, err := os.Open("./testdata/update_history.json")
	require.NoError(t, err)
	defer r.Close()

	entries, err := ParseWindowsUpdateHistory(r)
	require.NoError(t, err)

	installed := FilterInstalledHistory(entries)

	// Of the 6 history records: one failed (ResultCode 4) and one uninstall
	// (Operation 2) are dropped, and KB5034441 appears twice (de-duped). That
	// leaves KB5034441, KB5034763, and the Defender definition update.
	require.Len(t, installed, 3)

	kbs := make([]string, 0, len(installed))
	for _, e := range installed {
		kbs = append(kbs, ParseKBID(e.Title))
	}
	assert.Contains(t, kbs, "KB5034441")
	assert.Contains(t, kbs, "KB5034763")
	assert.Contains(t, kbs, "KB2267602")

	// the kept KB5034441 entry is the newest (first seen)
	for _, e := range installed {
		if ParseKBID(e.Title) == "KB5034441" {
			assert.Equal(t, "/Date(1705334400000)/", e.Date)
		}
	}
}

func TestParseWindowsUpdateHistory_SingleObject(t *testing.T) {
	// PowerShell ConvertTo-Json emits a bare object (not an array) when the
	// collection has exactly one element.
	single := `{"Title":"2024-01 Cumulative Update (KB5034441)","Date":"/Date(1705334400000)/","Operation":1,"ResultCode":2,"UpdateID":"11111111-1111-1111-1111-111111111111","Categories":["Security Updates"]}`
	entries, err := ParseWindowsUpdateHistory(strings.NewReader(single))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "KB5034441", ParseKBID(entries[0].Title))
}

func TestParseKBID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2024-01 Cumulative Update for Windows Server 2022 (KB5034441)", "KB5034441"},
		{"Security Intelligence Update ... - KB2267602 (Version 1.405.0.0)", "KB2267602"},
		{"kb1234567 lower case", "KB1234567"},
		{"Intel - Display - 27.20.100.9466", ""},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equalf(t, tt.want, ParseKBID(tt.input), "ParseKBID(%q)", tt.input)
	}
}

func TestClassifyUpdate(t *testing.T) {
	// a reported category wins
	assert.Equal(t, "Security Updates", ClassifyUpdate([]string{"Security Updates"}, "anything"))
	// empty categories fall back to title inference
	assert.Equal(t, "Update Rollups", ClassifyUpdate(nil, "2024-02 Cumulative Update for .NET Framework (KB5034763)"))
	assert.Equal(t, "Security Updates", ClassifyUpdate([]string{}, "2024-01 Security Update for Windows (KB5030219)"))
	assert.Equal(t, "Servicing Stack Updates", ClassifyUpdate(nil, "Servicing Stack Update for Windows (KB5031234)"))
	assert.Equal(t, "Drivers", ClassifyUpdate(nil, "Intel - Display driver"))
	assert.Equal(t, "", ClassifyUpdate(nil, "Some unrecognized update"))
}

func TestOperationName(t *testing.T) {
	assert.Equal(t, "Installation", OperationName(UpdateOperationInstallation))
	assert.Equal(t, "Uninstallation", OperationName(UpdateOperationUninstallation))
	assert.Equal(t, "", OperationName(0))
}
