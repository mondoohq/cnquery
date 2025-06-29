// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package kernel

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
)

func TestLsmodParser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/debian.toml", &inventory.Asset{})
	require.NoError(t, err)

	f, err := mock.RunCommand("/sbin/lsmod")
	require.NoError(t, err)

	entries := ParseLsmod(f.Stdout)
	assert.Equal(t, 40, len(entries))

	expected := &KernelModule{
		Name:   "cryptd",
		Size:   "24576",
		UsedBy: "3",
	}
	found := findModule(entries, "cryptd")
	assert.Equal(t, expected, found)
}

func TestLinuxProcModulesParser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/debian.toml", &inventory.Asset{})
	require.NoError(t, err)

	f, err := mock.FileSystem().Open("/proc/modules")
	require.NoError(t, err)
	defer f.Close()

	entries := ParseLinuxProcModules(f)
	assert.Equal(t, 40, len(entries))

	expected := &KernelModule{
		Name:   "cryptd",
		Size:   "24576",
		UsedBy: "3",
	}
	found := findModule(entries, "cryptd")
	assert.Equal(t, expected, found)
}

func TestKldstatParser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/freebsd12.toml", &inventory.Asset{})
	require.NoError(t, err)

	f, err := mock.RunCommand("kldstat")
	require.NoError(t, err)

	entries := ParseKldstat(f.Stdout)
	assert.Equal(t, 4, len(entries))

	expected := &KernelModule{
		Name:   "smbus.ko",
		Size:   "a30",
		UsedBy: "1",
	}
	found := findModule(entries, "smbus.ko")
	assert.Equal(t, expected, found)
}

func TestKextstatParser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/osx.toml", &inventory.Asset{})
	require.NoError(t, err)

	f, err := mock.RunCommand("kextstat")
	require.NoError(t, err)

	entries := ParseKextstat(f.Stdout)
	assert.Equal(t, 33, len(entries))

	expected := &KernelModule{
		Name:   "com.apple.kpi.mach",
		Size:   "0x62e0",
		UsedBy: "144",
	}
	found := findModule(entries, "com.apple.kpi.mach")
	assert.Equal(t, expected, found)
}

func TestLinuxSysModuleParser(t *testing.T) {
	// Create an in-memory filesystem to simulate /sys/module structure
	fs := afero.NewMemMapFs()

	// Create /sys/module directory structure with test modules
	err := fs.MkdirAll("/sys/module/cryptd", 0755)
	require.NoError(t, err)
	err = fs.MkdirAll("/sys/module/ext4", 0755)
	require.NoError(t, err)
	err = fs.MkdirAll("/sys/module/unloaded_module", 0755)
	require.NoError(t, err)

	// Create initstate files (live modules)
	err = afero.WriteFile(fs, "/sys/module/cryptd/initstate", []byte("live\n"), 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "/sys/module/ext4/initstate", []byte("live\n"), 0644)
	require.NoError(t, err)
	// Create an unloaded module (should be filtered out)
	err = afero.WriteFile(fs, "/sys/module/unloaded_module/initstate", []byte("going\n"), 0644)
	require.NoError(t, err)

	// Create coresize files
	err = afero.WriteFile(fs, "/sys/module/cryptd/coresize", []byte("24576\n"), 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "/sys/module/ext4/coresize", []byte("589824\n"), 0644)
	require.NoError(t, err)

	// Create refcnt files
	err = afero.WriteFile(fs, "/sys/module/cryptd/refcnt", []byte("3\n"), 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "/sys/module/ext4/refcnt", []byte("1\n"), 0644)
	require.NoError(t, err)

	// Parse the modules
	entries, err := ParseLinuxSysModule(fs)
	require.NoError(t, err)

	// Should only find 2 live modules (unloaded_module should be filtered out)
	assert.Equal(t, 2, len(entries))

	// Check cryptd module
	cryptd := findModule(entries, "cryptd")
	require.NotNil(t, cryptd)
	assert.Equal(t, "cryptd", cryptd.Name)
	assert.Equal(t, "24576", cryptd.Size)
	assert.Equal(t, "3", cryptd.UsedBy)

	// Check ext4 module
	ext4 := findModule(entries, "ext4")
	require.NotNil(t, ext4)
	assert.Equal(t, "ext4", ext4.Name)
	assert.Equal(t, "589824", ext4.Size)
	assert.Equal(t, "1", ext4.UsedBy)

	// Ensure unloaded module is not included
	unloaded := findModule(entries, "unloaded_module")
	assert.Nil(t, unloaded)
}

func TestLinuxSysModuleParserMissingFiles(t *testing.T) {
	// Test behavior when some files are missing
	fs := afero.NewMemMapFs()

	// Create module directory with only initstate
	err := fs.MkdirAll("/sys/module/minimal_module", 0755)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "/sys/module/minimal_module/initstate", []byte("live\n"), 0644)
	require.NoError(t, err)
	// No coresize or refcnt files

	entries, err := ParseLinuxSysModule(fs)
	require.NoError(t, err)
	assert.Equal(t, 1, len(entries))

	module := entries[0]
	assert.Equal(t, "minimal_module", module.Name)
	assert.Equal(t, "0", module.Size)    // Default when coresize is missing
	assert.Equal(t, "0", module.UsedBy)  // Default when refcnt is missing
}

func TestLinuxSysModuleParserNoSysModule(t *testing.T) {
	// Test behavior when /sys/module doesn't exist
	fs := afero.NewMemMapFs()

	entries, err := ParseLinuxSysModule(fs)
	// Should not error, but return empty list
	assert.NoError(t, err)
	assert.Equal(t, 0, len(entries))
}

func findModule(modules []*KernelModule, name string) *KernelModule {
	for i := range modules {
		if modules[i].Name == name {
			return modules[i]
		}
	}
	return nil
}
