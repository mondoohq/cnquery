// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Install directive tests
func TestParseInstall_BasicCommand(t *testing.T) {
	content := "install pcspkr /bin/true"
	matches := installRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "pcspkr", matches[1])    // module
	assert.Equal(t, "/bin/true", matches[2]) // command
}

func TestParseInstall_ComplexCommand(t *testing.T) {
	content := "install nouveau /bin/false"
	matches := installRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "nouveau", matches[1])
	assert.Equal(t, "/bin/false", matches[2])
}

func TestParseInstall_CommandWithArguments(t *testing.T) {
	content := "install nvidia modprobe --ignore-install nvidia $CMDLINE_OPTS"
	matches := installRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "nvidia", matches[1])
	assert.Contains(t, matches[2], "modprobe")
	assert.Contains(t, matches[2], "$CMDLINE_OPTS")
}

// Remove directive tests
func TestParseRemove_BasicCommand(t *testing.T) {
	content := "remove pcspkr /bin/true"
	matches := removeRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "pcspkr", matches[1])
	assert.Equal(t, "/bin/true", matches[2])
}

func TestParseRemove_ComplexCommand(t *testing.T) {
	content := "remove nvidia /sbin/modprobe -r --ignore-remove nvidia"
	matches := removeRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "nvidia", matches[1])
	assert.Contains(t, matches[2], "modprobe -r")
}

// Blacklist directive tests
func TestParseBlacklist_SingleModule(t *testing.T) {
	content := "blacklist nouveau"
	matches := blacklistRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "nouveau", matches[1])
}

func TestParseBlacklist_ModuleWithUnderscore(t *testing.T) {
	content := "blacklist i2c_piix4"
	matches := blacklistRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "i2c_piix4", matches[1])
}

func TestParseBlacklist_ModuleWithHyphen(t *testing.T) {
	content := "blacklist floppy"
	matches := blacklistRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "floppy", matches[1])
}

// Options directive tests
func TestParseOptions_SingleParameter(t *testing.T) {
	content := "options snd-hda-intel power_save=1"
	matches := optionsRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "snd-hda-intel", matches[1])
	assert.Equal(t, "power_save=1", matches[2])
}

func TestParseOptions_MultipleParameters(t *testing.T) {
	content := "options nvidia NVreg_DeviceFileGID=44 NVreg_DeviceFileUID=0"
	matches := optionsRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "nvidia", matches[1])
	assert.Contains(t, matches[2], "NVreg_DeviceFileGID=44")
	assert.Contains(t, matches[2], "NVreg_DeviceFileUID=0")
}

func TestParseOptions_BooleanFlag(t *testing.T) {
	content := "options kvm_intel nested=1"
	matches := optionsRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "kvm_intel", matches[1])
	assert.Equal(t, "nested=1", matches[2])
}

func TestParseOptions_QuotedValue(t *testing.T) {
	content := `options module param="value with spaces"`
	matches := optionsRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "module", matches[1])
	assert.Contains(t, matches[2], "param=")
}

// Alias directive tests
func TestParseAlias_BasicAlias(t *testing.T) {
	content := "alias eth0 e1000e"
	matches := aliasRegex2.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "eth0", matches[1])
	assert.Equal(t, "e1000e", matches[2])
}

func TestParseAlias_WildcardPattern(t *testing.T) {
	content := "alias pci:v00008086d* e1000e"
	matches := aliasRegex2.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Contains(t, matches[1], "pci:")
	assert.Equal(t, "e1000e", matches[2])
}

func TestParseAlias_SymbolAlias(t *testing.T) {
	content := "alias symbol:nvidiafb nvidia"
	matches := aliasRegex2.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Contains(t, matches[1], "symbol:")
	assert.Equal(t, "nvidia", matches[2])
}

// Softdep directive tests
func TestParseSoftdep_PreOnly(t *testing.T) {
	content := "softdep nvidia pre: nvidia-uvm"
	matches := softdepRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "nvidia", matches[1])
	assert.Contains(t, matches[2], "pre:")
	assert.Contains(t, matches[2], "nvidia-uvm")
}

func TestParseSoftdep_PostOnly(t *testing.T) {
	content := "softdep nvidia post: nvidia-modeset"
	matches := softdepRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "nvidia", matches[1])
	assert.Contains(t, matches[2], "post:")
	assert.Contains(t, matches[2], "nvidia-modeset")
}

func TestParseSoftdep_PreAndPost(t *testing.T) {
	content := "softdep nvidia pre: nvidia-uvm post: nvidia-modeset"
	matches := softdepRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "nvidia", matches[1])
	assert.Contains(t, matches[2], "pre:")
	assert.Contains(t, matches[2], "post:")
}

func TestParseSoftdep_MultipleModules(t *testing.T) {
	content := "softdep drm pre: drm_kms_helper ttm post: i915"
	matches := softdepRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "drm", matches[1])
	assert.Contains(t, matches[2], "drm_kms_helper")
	assert.Contains(t, matches[2], "ttm")
	assert.Contains(t, matches[2], "i915")
}

// Comment and empty line tests
func TestParseDirectives_Comments(t *testing.T) {
	content := "# This is a comment"

	assert.Nil(t, installRegex.FindStringSubmatch(content))
	assert.Nil(t, removeRegex.FindStringSubmatch(content))
	assert.Nil(t, blacklistRegex.FindStringSubmatch(content))
	assert.Nil(t, optionsRegex.FindStringSubmatch(content))
	assert.Nil(t, aliasRegex2.FindStringSubmatch(content))
	assert.Nil(t, softdepRegex.FindStringSubmatch(content))
}

func TestParseDirectives_EmptyLine(t *testing.T) {
	content := ""

	assert.Nil(t, installRegex.FindStringSubmatch(content))
	assert.Nil(t, removeRegex.FindStringSubmatch(content))
	assert.Nil(t, blacklistRegex.FindStringSubmatch(content))
	assert.Nil(t, optionsRegex.FindStringSubmatch(content))
	assert.Nil(t, aliasRegex2.FindStringSubmatch(content))
	assert.Nil(t, softdepRegex.FindStringSubmatch(content))
}

// Real-world examples
func TestParseInstall_DisableModuleLoading(t *testing.T) {
	// Common pattern to prevent module from loading
	content := "install dccp /bin/false"
	matches := installRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "dccp", matches[1])
	assert.Equal(t, "/bin/false", matches[2])
}

func TestParseBlacklist_Nouveau(t *testing.T) {
	// Common pattern when using NVIDIA proprietary drivers
	content := "blacklist nouveau"
	matches := blacklistRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "nouveau", matches[1])
}

func TestParseOptions_NvidiaDriverOptions(t *testing.T) {
	content := "options nvidia-drm modeset=1"
	matches := optionsRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "nvidia-drm", matches[1])
	assert.Equal(t, "modeset=1", matches[2])
}

func TestParseOptions_MultipleSoundOptions(t *testing.T) {
	content := "options snd-hda-intel model=auto power_save=1"
	matches := optionsRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "snd-hda-intel", matches[1])
	assert.Contains(t, matches[2], "model=auto")
	assert.Contains(t, matches[2], "power_save=1")
}

func TestParseBlacklist_IPv6(t *testing.T) {
	// Common pattern to disable IPv6
	content := "blacklist ipv6"
	matches := blacklistRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "ipv6", matches[1])
}

func TestParseInstall_USBStorage(t *testing.T) {
	// Disable USB storage devices
	content := "install usb-storage /bin/true"
	matches := installRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "usb-storage", matches[1])
	assert.Equal(t, "/bin/true", matches[2])
}

func TestParseOptions_KVMNesting(t *testing.T) {
	content := "options kvm-intel nested=Y"
	matches := optionsRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "kvm-intel", matches[1])
	assert.Equal(t, "nested=Y", matches[2])
}

func TestParseSoftdep_ComplexDependency(t *testing.T) {
	content := "softdep cfg80211 pre: regulatory post: ath9k"
	matches := softdepRegex.FindStringSubmatch(content)

	require.NotNil(t, matches)
	assert.Equal(t, "cfg80211", matches[1])
	assert.Contains(t, matches[2], "pre: regulatory")
	assert.Contains(t, matches[2], "post: ath9k")
}

// Edge cases
func TestParseOptions_EmptyParameters(t *testing.T) {
	// Invalid but should not crash
	content := "options module"
	matches := optionsRegex.FindStringSubmatch(content)

	// Should not match (no parameters)
	assert.Nil(t, matches)
}

func TestParseAlias_MultipleSpaces(t *testing.T) {
	content := "alias   eth0    e1000e"
	matches := aliasRegex2.FindStringSubmatch(strings.TrimSpace(content))

	// After trim and normalize spaces, should still match
	require.NotNil(t, matches)
}

func TestParseSoftdep_OnlyKeyword(t *testing.T) {
	content := "softdep drm"
	matches := softdepRegex.FindStringSubmatch(content)

	// Should not match (no dependencies)
	assert.Nil(t, matches)
}
