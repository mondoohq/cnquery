// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fixture holds a newer third-party installer run and a newer XProtect data
// blob than the last real macOS update. Both have to be ignored, or a Mac months
// behind on macOS reports as patched days ago.
func TestParseMacosInstallHistory(t *testing.T) {
	f, err := os.Open("./testdata/InstallHistory.plist")
	require.NoError(t, err)
	defer f.Close()

	got, ok, err := ParseMacosInstallHistory(f)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "2026-07-23T10:48:32Z", got.Format(time.RFC3339))
}

func macosPlist(entries string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<array>` + entries + `
</array>
</plist>`
}

func macosEntry(date, name, process, contentType string) string {
	entry := `
	<dict>
		<key>date</key>
		<date>` + date + `</date>
		<key>displayName</key>
		<string>` + name + `</string>
		<key>processName</key>
		<string>` + process + `</string>`
	if contentType != "" {
		entry += `
		<key>contentType</key>
		<string>` + contentType + `</string>`
	}
	return entry + `
	</dict>`
}

func TestParseMacosInstallHistoryFiltering(t *testing.T) {
	tests := []struct {
		name    string
		entries string
		want    string
	}{
		{
			name:    "softwareupdated os update",
			entries: macosEntry("2026-07-23T10:48:32Z", "macOS 26.5.2", "softwareupdated", ""),
			want:    "2026-07-23T10:48:32Z",
		},
		{
			name:    "softwareupdate cli install",
			entries: macosEntry("2026-02-18T17:39:44Z", "RosettaUpdateAuto", "softwareupdate", ""),
			want:    "2026-02-18T17:39:44Z",
		},
		{
			name:    "config data is excluded",
			entries: macosEntry("2026-08-19T07:01:03Z", "XProtectPlistConfigData", "softwareupdated", "config-data"),
			want:    "",
		},
		{
			name:    "third party installer is excluded",
			entries: macosEntry("2026-08-25T11:29:02Z", "Mondoo", "installer", ""),
			want:    "",
		},
		{
			name:    "rapid security response counts",
			entries: macosEntry("2026-03-19T00:23:54Z", "macOS Rapid Security Response 26.3.1 (a)", "softwareupdated", ""),
			want:    "2026-03-19T00:23:54Z",
		},
		{
			name: "newest qualifying entry wins regardless of order",
			entries: macosEntry("2026-07-23T10:48:32Z", "macOS 26.5.2", "softwareupdated", "") +
				macosEntry("2026-03-17T07:18:17Z", "macOS 26.3.1", "softwareupdated", "") +
				macosEntry("2026-08-19T07:01:03Z", "XProtectPlistConfigData", "softwareupdated", "config-data"),
			want: "2026-07-23T10:48:32Z",
		},
		{
			name:    "empty history",
			entries: "",
			want:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok, err := ParseMacosInstallHistory(strings.NewReader(macosPlist(test.entries)))
			require.NoError(t, err)
			if test.want == "" {
				assert.False(t, ok)
				assert.True(t, got.IsZero())
				return
			}
			require.True(t, ok)
			assert.Equal(t, test.want, got.Format(time.RFC3339))
		})
	}
}

func TestParseMacosInstallHistoryMalformed(t *testing.T) {
	_, ok, err := ParseMacosInstallHistory(strings.NewReader("not a plist"))
	assert.Error(t, err, "a malformed plist must be reported, not read as an absent record")
	assert.False(t, ok)
}

// A command's stdout does not seek, and the plist decoder needs it to. Feeding
// it one must work rather than fail on the seek.
func TestParseMacosInstallHistoryNonSeekableReader(t *testing.T) {
	plist := macosPlist(macosEntry("2026-07-23T10:48:32Z", "macOS 26.5.2", "softwareupdated", ""))

	got, ok, err := ParseMacosInstallHistory(nonSeekableReader{strings.NewReader(plist)})
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "2026-07-23T10:48:32Z", got.Format(time.RFC3339))
}

type nonSeekableReader struct{ r *strings.Reader }

func (n nonSeekableReader) Read(p []byte) (int, error) { return n.r.Read(p) }
