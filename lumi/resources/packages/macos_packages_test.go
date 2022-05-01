package packages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/packages"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestMacOsXPackageParser(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/packages_macos.toml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("system_profiler SPApplicationsDataType -xml")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := packages.ParseMacOSPackages(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(m), "detected the right amount of packages")

	assert.Equal(t, "Preview", m[0].Name, "pkg name detected")
	assert.Equal(t, "10.0", m[0].Version, "pkg version detected")
	assert.Equal(t, packages.MacosPkgFormat, m[0].Format, "pkg format detected")

	assert.Equal(t, "Contacts", m[1].Name, "pkg name detected")
	assert.Equal(t, "11.0", m[1].Version, "pkg version detected")
	assert.Equal(t, packages.MacosPkgFormat, m[0].Format, "pkg format detected")
}
