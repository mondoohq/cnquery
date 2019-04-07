package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	mock "go.mondoo.io/mondoo/motor/mock/toml"
	"go.mondoo.io/mondoo/motor/types"
)

func TestApkUpdateParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "updates_apk.toml"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("apk version -v -l '<'")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := ParseApkUpdates(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(m), "detected the right amount of package updates")

	assert.Equal(t, "busybox", m[0].Name, "pkg name detected")
	assert.Equal(t, "1.28.4-r0", m[0].Version, "pkg version detected")
	assert.Equal(t, "1.28.4-r1", m[0].Available, "pkg available version detected")

	assert.Equal(t, "ssl_client", m[1].Name, "pkg name detected")
	assert.Equal(t, "1.28.4-r0", m[1].Version, "pkg version detected")
	assert.Equal(t, "1.28.4-r1", m[0].Available, "pkg available version detected")
}

func TestDpkgUpdateParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "updates_dpkg.toml"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("DEBIAN_FRONTEND=noninteractive apt-get upgrade --dry-run")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := ParseDpkgUpdates(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 13, len(m), "detected the right amount of package updates")

	assert.Equal(t, "base-files", m[0].Name, "pkg name detected")
	assert.Equal(t, "10.1ubuntu2", m[0].Version, "pkg version detected")
	assert.Equal(t, "10.1ubuntu2.1", m[0].Available, "pkg available version detected")

	assert.Equal(t, "ncurses-bin", m[1].Name, "pkg name detected")
	assert.Equal(t, "6.1-1ubuntu1", m[1].Version, "pkg version detected")
	assert.Equal(t, "10.1ubuntu2.1", m[0].Available, "pkg available version detected")
}

func TestRpmUpdateParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "updates_rpm.toml"})
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

	assert.Equal(t, "python-libs", m[0].Name, "pkg name detected")
	assert.Equal(t, "", m[0].Version, "pkg version detected")
	assert.Equal(t, "0:2.7.5-69.el7_5", m[0].Available, "pkg available version detected")

	assert.Equal(t, "binutils", m[1].Name, "pkg name detected")
	assert.Equal(t, "", m[1].Version, "pkg version detected")
	assert.Equal(t, "0:2.7.5-69.el7_5", m[0].Available, "pkg available version detected")
}

func TestZypperUpdateParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "updates_zypper.toml"})
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

	assert.Equal(t, "aaa_base", m[0].Name, "pkg name detected")
	assert.Equal(t, "13.2+git20140911.61c1681-28.3.1", m[0].Version, "pkg version detected")

	assert.Equal(t, "bash", m[1].Name, "pkg name detected")
	assert.Equal(t, "4.3-83.3.1", m[1].Version, "pkg version detected")
}

// SUSE OS updates
func TestZypperPatchParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "updates_zypper.toml"})
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
