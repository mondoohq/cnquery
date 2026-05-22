// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWindowsUpdates(t *testing.T) {
	r, err := os.Open("./testdata/updates_win.json")
	require.NoError(t, err)
	defer r.Close()

	updates, err := ParseWindowsUpdates(r)
	require.NoError(t, err)
	assert.Equal(t, 6, len(updates), "detected the right amount of updates")

	u, err := findUpdate(updates, "4537759")
	require.NoError(t, err)
	assert.Equal(t, "2020-02 Security Update for Adobe Flash Player for Windows Server 2019 for x64-based Systems (KB4537759)", u.Title)
	assert.Equal(t, "Critical", u.MsrcSeverity)
	assert.Equal(t, "https://support.microsoft.com/help/4537759", u.SupportUrl)
	assert.Equal(t, []string{"CVE-2020-0001"}, u.CveIDs)
	assert.Equal(t, []string{"Security Updates"}, u.Categories)

	// empty input
	updates, err = ParseWindowsUpdates(strings.NewReader(""))
	require.NoError(t, err)
	assert.Equal(t, 0, len(updates))
}

func TestParseWindowsUpdates_SingleObject(t *testing.T) {
	// ConvertTo-Json emits a bare object when the search returns one update.
	single := `{"UpdateID":"abc","Title":"Update (KB4538461)","MsrcSeverity":"Important","KBArticleIDs":["4538461"],"RebootRequired":true}`
	updates, err := ParseWindowsUpdates(strings.NewReader(single))
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.Equal(t, "4538461", updates[0].KBArticleIDs[0])
	assert.True(t, updates[0].RebootRequired)
}

func TestWindowsUpdateToOperatingSystemUpdate(t *testing.T) {
	u := WindowsUpdate{
		UpdateID:       "0b669361-bcf2-4c1c-b4fb-e5629cfdc3c0",
		Title:          "2020-02 Security Update (KB4537759)",
		MsrcSeverity:   "Critical",
		RebootRequired: true,
		KBArticleIDs:   []string{"4537759"},
	}
	osUpdate, ok := u.toOperatingSystemUpdate()
	require.True(t, ok)
	assert.Equal(t, "4537759", osUpdate.Name, "KB becomes the os.update name")
	assert.Equal(t, "0b669361-bcf2-4c1c-b4fb-e5629cfdc3c0", osUpdate.ID)
	assert.Equal(t, "2020-02 Security Update (KB4537759)", osUpdate.Description)
	assert.Equal(t, "Critical", osUpdate.Severity)
	assert.Equal(t, "windows/updates", osUpdate.Format)
	assert.True(t, osUpdate.Restart)

	// updates without a KB are dropped from os.update
	_, ok = WindowsUpdate{UpdateID: "x", Title: "Driver update"}.toOperatingSystemUpdate()
	assert.False(t, ok)
}

func findUpdate(updates []WindowsUpdate, kb string) (WindowsUpdate, error) {
	for i := range updates {
		if slices.Contains(updates[i].KBArticleIDs, kb) {
			return updates[i], nil
		}
	}
	return WindowsUpdate{}, errors.New("not found")
}

// findKb is shared with the other update-manager tests in this package.
func findKb(pkgs []OperatingSystemUpdate, name string) (OperatingSystemUpdate, error) {
	for i := range pkgs {
		if pkgs[i].Name == name {
			return pkgs[i], nil
		}
	}
	return OperatingSystemUpdate{}, errors.New("not found")
}
