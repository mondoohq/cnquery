package packages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestRpmUpdateParser(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: "./testdata/updates_rpm.toml"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("python")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := ParseRpmUpdates(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 8, len(m), "detected the right amount of package updates")

	update := m["python-libs"]
	assert.Equal(t, "python-libs", update.Name, "pkg name detected")
	assert.Equal(t, "", update.Version, "pkg version detected")
	assert.Equal(t, "0:2.7.5-69.el7_5", update.Available, "pkg available version detected")

	update = m["binutils"]
	assert.Equal(t, "binutils", update.Name, "pkg name detected")
	assert.Equal(t, "", update.Version, "pkg version detected")
	assert.Equal(t, "0:2.27-28.base.el7_5.1", update.Available, "pkg available version detected")
}

func TestZypperUpdateParser(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: "./testdata/updates_zypper.toml"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("zypper --xmlout list-updates")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := ParseZypperUpdates(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 22, len(m), "detected the right amount of package updates")

	update := m["aaa_base"]
	assert.Equal(t, "aaa_base", update.Name, "pkg name detected")
	assert.Equal(t, "13.2+git20140911.61c1681-28.3.1", update.Version, "pkg version detected")

	update = m["bash"]
	assert.Equal(t, "bash", update.Name, "pkg name detected")
	assert.Equal(t, "4.3-83.3.1", update.Version, "pkg version detected")
}

// SUSE OS updates
func TestZypperPatchParser(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: "./testdata/updates_zypper.toml"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("zypper --xmlout list-updates -t patch")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := ParseZypperPatches(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(m), "detected the right amount of packages")

	assert.Equal(t, "openSUSE-2018-397", m[0].Name, "update name detected")
	assert.Equal(t, "moderate", m[0].Severity, "severity version detected")

}
