// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
)

func TestDpkgParser(t *testing.T) {
	pf := &inventory.Platform{
		Name:    "ubuntu",
		Version: "18.04",
		Arch:    "x86_64",
		Family:  []string{"debian", "linux", "unix", "os"},
		Labels: map[string]string{
			"distro-id": "ubuntu",
		},
	}

	mock, err := mock.New(0, "./testdata/packages_dpkg.toml", &inventory.Asset{})
	require.NoError(t, err)
	f, err := mock.FileSystem().Open("/var/lib/dpkg/status")
	require.NoError(t, err)
	defer f.Close()

	m, err := ParseDpkgPackages(pf, f)
	require.NoError(t, err)
	assert.Equal(t, 10, len(m), "detected the right amount of packages")

	p := Package{
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
		PUrl:   "pkg:deb/ubuntu/fdisk@2.31.1-0.4ubuntu3.1?arch=amd64&distro=ubuntu-18.04",
		CPEs:   []string{"cpe:2.3:a:fdisk:fdisk:2.31.1-0.4ubuntu3.1:amd64:*:*:*:*:*:*"},
		Format: "deb",
	}
	assert.Equal(t, findPkg(m, p.Name), p, p.Name)

	p = Package{
		Name:    "libaudit1",
		Version: "1:2.4-1+b1",
		Arch:    "amd64",
		Status:  "install ok installed",
		Origin:  "audit (1:2.4-1)",
		Description: `Dynamic library for security auditing
The audit-libs package contains the dynamic libraries needed for
applications to use the audit framework. It is used to monitor systems for
security related events.`,
		PUrl:           "pkg:deb/ubuntu/libaudit1@1%3A2.4-1%2Bb1?arch=amd64&distro=ubuntu-18.04",
		CPEs:           []string{"cpe:2.3:a:libaudit1:libaudit1:1:amd64:*:*:*:*:*:*"},
		Format:         "deb",
		FilesAvailable: PkgFilesAsync,
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	p = Package{
		Name:    "libss2",
		Version: "1.44.1-1",
		Arch:    "amd64",
		Status:  "install ok installed",
		Origin:  "e2fsprogs",
		Description: `command-line interface parsing library
libss provides a simple command-line interface parser which will
accept input from the user, parse the command into an argv argument
vector, and then dispatch it to a handler function.
.
It was originally inspired by the Multics SubSystem library.`,
		PUrl:           "pkg:deb/ubuntu/libss2@1.44.1-1?arch=amd64&distro=ubuntu-18.04",
		CPEs:           []string{"cpe:2.3:a:libss2:libss2:1.44.1-1:amd64:*:*:*:*:*:*"},
		Format:         "deb",
		FilesAvailable: PkgFilesAsync,
	}
	assert.Equal(t, findPkg(m, p.Name), p, p.Name)

	// fetch package files
	mgr := &DebPkgManager{
		conn:     mock,
		platform: pf,
	}
	pkgFiles, err := mgr.Files(p.Name, p.Version, p.Arch)
	require.NoError(t, err)
	assert.Equal(t, 11, len(pkgFiles), "detected the right amount of package files")
	assert.Contains(t, pkgFiles, FileRecord{Path: "/lib/aarch64-linux-gnu/libss.so.2.0"})
	assert.Contains(t, pkgFiles, FileRecord{Path: "/lib/aarch64-linux-gnu/libss.so.2"})
}

func TestDpkgParserStatusD(t *testing.T) {
	pf := &inventory.Platform{
		Name:    "ubuntu",
		Version: "18.04",
		Arch:    "x86_64",
		Family:  []string{"debian", "linux", "unix", "os"},
		Labels: map[string]string{
			"distro-id": "ubuntu",
		},
	}

	mock, err := mock.New(0, "./testdata/packages_dpkg_statusd.toml", &inventory.Asset{})
	require.NoError(t, err)
	f, err := mock.FileSystem().Open("/var/lib/dpkg/status.d/base")
	require.NoError(t, err)
	defer f.Close()

	m, err := ParseDpkgPackages(pf, f)
	require.NoError(t, err)
	assert.Equal(t, 1, len(m), "detected the right amount of packages")

	p := Package{
		Name:    "base-files",
		Version: "9.9+deb9u11",
		Arch:    "amd64",
		Description: `Debian base system miscellaneous files
This package contains the basic filesystem hierarchy of a Debian system, and
several important miscellaneous files, such as /etc/debian_version,
/etc/host.conf, /etc/issue, /etc/motd, /etc/profile, and others,
and the text of several common licenses in use on Debian systems.`,
		PUrl:   "pkg:deb/ubuntu/base-files@9.9%2Bdeb9u11?arch=amd64&distro=ubuntu-18.04",
		CPEs:   []string{"cpe:2.3:a:base-files:base-files:9.9\\+deb9u11:amd64:*:*:*:*:*:*"},
		Format: "deb",
	}
	assert.Contains(t, m, p, "fdisk detected")
}

func TestDpkgUpdateParser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/updates_dpkg.toml", &inventory.Asset{})
	require.NoError(t, err)
	c, err := mock.RunCommand("DEBIAN_FRONTEND=noninteractive apt-get upgrade --dry-run")
	require.NoError(t, err)
	assert.Nil(t, err)

	m, err := ParseDpkgUpdates(c.Stdout)
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
