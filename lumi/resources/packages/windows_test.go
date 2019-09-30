package packages

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestPowershellEncoding(t *testing.T) {
	expected := "powershell.exe -EncodedCommand ZABpAHIAIAAiAGMAOgBcAHAAcgBvAGcAcgBhAG0AIABmAGkAbABlAHMAIgAgAA=="
	cmd := string("dir \"c:\\program files\" ")
	assert.Equal(t, expected, EncodePowershell(cmd))
}

func TestWindowsAppxPackagesParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/windows_2019.toml"})
	if err != nil {
		t.Fatal(err)
	}

	c, err := mock.RunCommand(fmt.Sprintf("powershell -c \"%s\"", WINDOWS_QUERY_APPX_PACKAGES))
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
		Arch:    "noarch",
		Format:  "windows/appx",
	}
	assert.Contains(t, m, p)

}

func TestWindowsHotFixParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/windows_2019.toml"})
	if err != nil {
		t.Fatal(err)
	}

	c, err := mock.RunCommand(fmt.Sprintf("powershell -c \"%s\"", WINDOWS_QUERY_HOTFIXES))
	if err != nil {
		t.Fatal(err)
	}

	m, err := ParseWindowsHotfixes(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(m), "detected the right amount of packages")

	var p Package
	p = Package{
		Name:        "KB4486553",
		Description: "Update",
		Format:      "windows/hotfix",
	}
	assert.Contains(t, m, p)

}

func TestWinOSUpdatesParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/windows_2019.toml"})
	if err != nil {
		t.Fatal(err)
	}

	cmd := EncodePowershell(WINDOWS_QUERY_WSUS_AVAILABLE)
	c, err := mock.RunCommand(cmd)
	if err != nil {
		t.Fatal(err)
	}

	m, err := ParseWindowsUpdates(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(m), "detected the right amount of packages")

	assert.Equal(t, "2267602", m[0].Name, "update id detected")
	assert.Equal(t, "Definition Update for Windows Defender Antivirus - KB2267602 (Definition 1.289.646.0)", m[0].Description, "update title detected")

	assert.Equal(t, "4487044", m[1].Name, "update id detected")
	assert.Equal(t, "2019-02 Cumulative Update for Windows Server 2019 (1809) for x64-based Systems (KB4487044)", m[1].Description, "update title detected")
}
