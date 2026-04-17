// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package wordpress

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanPluginDir(t *testing.T) {
	afs := &afero.Afero{Fs: afero.NewOsFs()}
	plugins, err := ScanPluginDir(afs, "./testdata")
	require.NoError(t, err)

	// akismet + contact-form-7 = 2 plugins
	// not-a-plugin is skipped (no readme.txt)
	assert.Equal(t, 2, len(plugins))

	var akismet, cf7 *WordPressPlugin
	for i := range plugins {
		switch plugins[i].Slug {
		case "akismet":
			akismet = &plugins[i]
		case "contact-form-7":
			cf7 = &plugins[i]
		}
	}

	require.NotNil(t, akismet)
	assert.Equal(t, "akismet", akismet.Slug)
	assert.Equal(t, "5.3.3", akismet.Version)
	assert.Equal(t, "Akismet Anti-spam: Spam Protection", akismet.DisplayName)
	assert.Equal(t, "GPLv2 or later", akismet.License)
	assert.Equal(t, "5.8", akismet.RequiresWp)
	assert.Equal(t, "6.6", akismet.TestedUpTo)

	require.NotNil(t, cf7)
	assert.Equal(t, "contact-form-7", cf7.Slug)
	assert.Equal(t, "5.9.8", cf7.Version)
	assert.Equal(t, "Contact Form 7", cf7.DisplayName)
	assert.Equal(t, "6.2", cf7.RequiresWp)
}

func TestExtractPluginName(t *testing.T) {
	assert.Equal(t, "Akismet Anti-spam: Spam Protection", extractPluginName("=== Akismet Anti-spam: Spam Protection ==="))
	assert.Equal(t, "Hello", extractPluginName("=== Hello ==="))
	assert.Equal(t, "", extractPluginName("Not a plugin name"))
	assert.Equal(t, "", extractPluginName(""))
}

func TestNewPackageUrl(t *testing.T) {
	assert.Equal(t, "pkg:wordpress-plugin/akismet@5.3.3", NewPackageUrl("akismet", "5.3.3"))
}
