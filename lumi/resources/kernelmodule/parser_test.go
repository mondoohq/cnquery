package kernelmodule_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
	"go.mondoo.io/mondoo/lumi/resources/kernelmodule"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestLsmodParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/debian.toml"})
	require.NoError(t, err)

	f, err := mock.RunCommand("lsmod")
	require.NoError(t, err)

	entries := kernelmodule.ParseLsmod(f.Stdout)
	assert.Equal(t, 40, len(entries))

	expected := &kernelmodule.KernelModule{
		Name:   "cryptd",
		Size:   "24576",
		UsedBy: "3",
	}
	found := findModule(entries, "cryptd")
	assert.Equal(t, expected, found)
}

func TestLinuxProcModulesParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/debian.toml"})
	require.NoError(t, err)

	f, err := mock.File("/proc/modules")
	require.NoError(t, err)
	defer f.Close()

	entries := kernelmodule.ParseLinuxProcModules(f)
	assert.Equal(t, 40, len(entries))

	expected := &kernelmodule.KernelModule{
		Name:   "cryptd",
		Size:   "24576",
		UsedBy: "3",
	}
	found := findModule(entries, "cryptd")
	assert.Equal(t, expected, found)
}

func TestKldstatParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/freebsd12.toml"})
	require.NoError(t, err)

	f, err := mock.RunCommand("kldstat")
	require.NoError(t, err)

	entries := kernelmodule.ParseKldstat(f.Stdout)
	assert.Equal(t, 4, len(entries))

	expected := &kernelmodule.KernelModule{
		Name:   "smbus.ko",
		Size:   "a30",
		UsedBy: "1",
	}
	found := findModule(entries, "smbus.ko")
	assert.Equal(t, expected, found)
}

func TestKextstatParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/osx.toml"})
	require.NoError(t, err)

	f, err := mock.RunCommand("kextstat")
	require.NoError(t, err)

	entries := kernelmodule.ParseKextstat(f.Stdout)
	assert.Equal(t, 33, len(entries))

	expected := &kernelmodule.KernelModule{
		Name:   "com.apple.kpi.mach",
		Size:   "0x62e0",
		UsedBy: "144",
	}
	found := findModule(entries, "com.apple.kpi.mach")
	assert.Equal(t, expected, found)
}

func findModule(modules []*kernelmodule.KernelModule, name string) *kernelmodule.KernelModule {
	for i := range modules {
		if modules[i].Name == name {
			return modules[i]
		}
	}
	return nil
}
