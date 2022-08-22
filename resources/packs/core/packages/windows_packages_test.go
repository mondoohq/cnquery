package packages

import (
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers/mock"
	"go.mondoo.io/mondoo/motor/providers/os/powershell"
)

func TestWindowsAppxPackagesParser(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/windows_2019.toml")
	if err != nil {
		t.Fatal(err)
	}

	c, err := mock.RunCommand(powershell.Wrap(WINDOWS_QUERY_APPX_PACKAGES))
	if err != nil {
		t.Fatal(err)
	}

	m, err := ParseWindowsAppxPackages(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 28, len(m), "detected the right amount of packages")

	var p Package
	p = Package{
		Name:    "Microsoft.Windows.Cortana",
		Version: "1.11.5.17763",
		Arch:    "neutral",
		Format:  "windows/appx",
	}
	assert.Contains(t, m, p)

	// check empty return
	m, err = ParseWindowsAppxPackages(strings.NewReader(""))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(m), "detected the right amount of packages")
}

func TestWindowsHotFixParser(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/windows_2019.toml")
	if err != nil {
		t.Fatal(err)
	}

	c, err := mock.RunCommand(powershell.Wrap(WINDOWS_QUERY_HOTFIXES))
	if err != nil {
		t.Fatal(err)
	}

	hotfixes, err := ParseWindowsHotfixes(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(hotfixes), "detected the right amount of packages")

	timestamp := hotfixes[0].InstalledOnTime()
	assert.NotNil(t, timestamp)

	pkgs := HotFixesToPackages(hotfixes)
	var p Package
	p = Package{
		Name:        "KB4486553",
		Description: "Update",
		Format:      "windows/hotfix",
	}
	assert.Contains(t, pkgs, p)

	// check empty return
	hotfixes, err = ParseWindowsHotfixes(strings.NewReader(""))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(hotfixes), "detected the right amount of packages")
}

func TestWinOSUpdatesParser(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/windows_2019.toml")
	if err != nil {
		t.Fatal(err)
	}

	cmd := powershell.Encode(WINDOWS_QUERY_WSUS_AVAILABLE)
	c, err := mock.RunCommand(cmd)
	if err != nil {
		t.Fatal(err)
	}

	m, err := ParseWindowsUpdates(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(m), "detected the right amount of packages")

	pkg, err := findKb(m, "890830")
	require.NoError(t, err)
	assert.Equal(t, "890830", pkg.Name, "update id detected")
	assert.Equal(t, "Windows Malicious Software Removal Tool x64 - March 2020 (KB890830)", pkg.Description, "update title detected")

	pkg, err = findKb(m, "4538461")
	require.NoError(t, err)
	assert.Equal(t, "4538461", pkg.Name, "update id detected")
	assert.Equal(t, "2020-03 Cumulative Update for Windows Server 2019 (1809) for x64-based Systems (KB4538461)", pkg.Description, "update title detected")

	// check empty return
	m, err = ParseWindowsUpdates(strings.NewReader(""))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(m), "detected the right amount of packages")
}

func findKb(pkgs []Package, name string) (Package, error) {
	for i := range pkgs {
		if pkgs[i].Name == name {
			return pkgs[i], nil
		}
	}

	return Package{}, errors.New("not found")
}
