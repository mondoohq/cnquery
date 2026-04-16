// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSafariExtensionStates(t *testing.T) {
	// Safari Extensions.plist is a binary plist, but for testing we use XML format
	// which the plist library also supports
	plistXML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>com.agilebits.onepassword-safari</key>
	<dict>
		<key>Enabled</key>
		<true/>
	</dict>
	<key>com.example.disabled-ext</key>
	<dict>
		<key>Enabled</key>
		<false/>
	</dict>
	<key>com.example.no-enabled-key</key>
	<dict>
		<key>SomeOtherKey</key>
		<string>value</string>
	</dict>
</dict>
</plist>`

	fs := afero.NewMemMapFs()
	afs := &afero.Afero{Fs: fs}

	homeDir := "/Users/testuser"
	plistPath := homeDir + "/Library/Safari/Extensions/Extensions.plist"
	require.NoError(t, fs.MkdirAll(homeDir+"/Library/Safari/Extensions", 0755))
	require.NoError(t, afs.WriteFile(plistPath, []byte(plistXML), 0644))

	states := readSafariExtensionStates(afs, homeDir)

	// Enabled extension
	enabled, ok := states["com.agilebits.onepassword-safari"]
	assert.True(t, ok)
	assert.True(t, enabled)

	// Disabled extension
	disabled, ok := states["com.example.disabled-ext"]
	assert.True(t, ok)
	assert.False(t, disabled)

	// Extension without Enabled key should not be in map
	_, ok = states["com.example.no-enabled-key"]
	assert.False(t, ok)
}

func TestReadSafariExtensionStatesNonExistent(t *testing.T) {
	fs := afero.NewMemMapFs()
	afs := &afero.Afero{Fs: fs}

	states := readSafariExtensionStates(afs, "/nonexistent/user")
	assert.Empty(t, states)
}
