package packages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/packages"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestDpkgParser(t *testing.T) {
	mock, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: "./testdata/packages_dpkg.toml"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.File("/var/lib/dpkg/status")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	m, err := packages.ParseDpkgPackages(f)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 10, len(m), "detected the right amount of packages")

	var p packages.Package
	p = packages.Package{
		Name:    "fdisk",
		Version: "2.31.1-0.4ubuntu3.1",
		Arch:    "amd64",
		Status:  "install ok installed",
		Origin:  "util-linux",
		Description: `collection of partitioning utilities
This package contains the classic fdisk, sfdisk and cfdisk partitioning
utilities from the util-linux suite.
.
The utilities included in this package allow you to partition
your hard disk. The utilities supports both modern and legacy
partition tables (eg. GPT, MBR, etc).
.
The fdisk utility is the classical text-mode utility.
The cfdisk utilitity gives a more userfriendly curses based interface.
The sfdisk utility is mostly for automation and scripting uses.`,
		Format: "deb",
	}
	assert.Contains(t, m, p, "fdisk detected")

	p = packages.Package{
		Name:    "libaudit1",
		Version: "1:2.4-1+b1",
		Arch:    "amd64",
		Status:  "install ok installed",
		Origin:  "audit",
		Description: `Dynamic library for security auditing
The audit-libs package contains the dynamic libraries needed for
applications to use the audit framework. It is used to monitor systems for
security related events.`,
		Format: "deb",
	}
	assert.Contains(t, m, p, "libaudit1 detected")
}

func TestDpkgParserStatusD(t *testing.T) {
	mock, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: "./testdata/packages_dpkg_statusd.toml"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.File("/var/lib/dpkg/status.d/base")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	m, err := packages.ParseDpkgPackages(f)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(m), "detected the right amount of packages")

	var p packages.Package
	p = packages.Package{
		Name:    "base-files",
		Version: "9.9+deb9u11",
		Arch:    "amd64",
		Description: `Debian base system miscellaneous files
This package contains the basic filesystem hierarchy of a Debian system, and
several important miscellaneous files, such as /etc/debian_version,
/etc/host.conf, /etc/issue, /etc/motd, /etc/profile, and others,
and the text of several common licenses in use on Debian systems.`,
		Format: "deb",
	}
	assert.Contains(t, m, p, "fdisk detected")
}

func TestDpkgUpdateParser(t *testing.T) {
	mock, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: "./testdata/updates_dpkg.toml"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("DEBIAN_FRONTEND=noninteractive apt-get upgrade --dry-run")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := packages.ParseDpkgUpdates(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 13, len(m), "detected the right amount of package updates")

	update := m["base-files"]
	assert.Equal(t, "base-files", update.Name, "pkg name detected")
	assert.Equal(t, "10.1ubuntu2", update.Version, "pkg version detected")
	assert.Equal(t, "10.1ubuntu2.1", update.Available, "pkg available version detected")

	update = m["ncurses-bin"]
	assert.Equal(t, "ncurses-bin", update.Name, "pkg name detected")
	assert.Equal(t, "6.1-1ubuntu1", update.Version, "pkg version detected")
	assert.Equal(t, "6.1-1ubuntu1.18.04", update.Available, "pkg available version detected")
}
