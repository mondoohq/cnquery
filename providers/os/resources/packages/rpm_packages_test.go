// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages_test

import (
	"bytes"
	"fmt"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"io"
	"os"
	"path/filepath"
	"testing"

	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/resources/packages"
)

func TestRedhat7Parser(t *testing.T) {
	mock, err := mock.New("./testdata/packages_redhat7.toml", nil)
	if err != nil {
		t.Fatal(err)
	}

	pf := &inventory.Platform{
		Name:    "redhat",
		Version: "7.4",
		Arch:    "x86_64",
		Family:  []string{"redhat", "linux", "unix", "os"},
		Labels: map[string]string{
			"distro-id": "rhel",
		},
	}

	c, err := mock.RunCommand("rpm -qa --queryformat '%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE} %{ARCH} %{SUMMARY}\\n'")
	if err != nil {
		t.Fatal(err)
	}

	m := packages.ParseRpmPackages(pf, c.Stdout)
	assert.Equal(t, 144, len(m), "detected the right amount of packages")

	var p packages.Package
	p = packages.Package{
		Name:        "ncurses-base",
		Version:     "5.9-14.20130511.el7_4",
		Arch:        "noarch",
		Description: "Descriptions of common terminals",
		PUrl:        "pkg:rpm/rhel/ncurses-base@5.9-14.20130511.el7_4?arch=noarch&distro=rhel-7.4",
		CPE:         "cpe:2.3:a:ncurses-base:ncurses-base:5.9-14.20130511.el7_4:noarch:*:*:*:*:*:*",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "ncurses-base")

	p = packages.Package{
		Name:        "libstdc++",
		Version:     "4.8.5-28.el7_5.1",
		Arch:        "x86_64",
		Description: "GNU Standard C++ Library",
		PUrl:        "pkg:rpm/rhel/libstdc%2B%2B@4.8.5-28.el7_5.1?arch=x86_64&distro=rhel-7.4",
		CPE:         "cpe:2.3:a:libstdc++:libstdc\\+\\+:4.8.5-28.el7_5.1:x86_64:*:*:*:*:*:*",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "libstdc detected")

	p = packages.Package{
		Name:        "iputils",
		Version:     "20160308-10.el7",
		Arch:        "x86_64",
		Description: "Network monitoring tools including ping",
		PUrl:        "pkg:rpm/rhel/iputils@20160308-10.el7?arch=x86_64&distro=rhel-7.4",
		CPE:         "cpe:2.3:a:iputils:iputils:20160308-10.el7:x86_64:*:*:*:*:*:*",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "gpg-pubkey detected")

	p = packages.Package{
		Name:        "openssl-libs",
		Version:     "1:1.0.2k-12.el7",
		Epoch:       "1",
		Arch:        "x86_64",
		Description: "A general purpose cryptography library with TLS implementation",
		PUrl:        "pkg:rpm/rhel/openssl-libs@1%3A1.0.2k-12.el7?arch=x86_64&distro=rhel-7.4&epoch=1",
		CPE:         "cpe:2.3:a:openssl-libs:openssl-libs:1:x86_64:*:*:*:*:1:*",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "gpg-pubkey detected")

	p = packages.Package{
		Name:        "dbus-libs",
		Version:     "1:1.10.24-7.el7",
		Epoch:       "1",
		Arch:        "x86_64",
		Description: "Libraries for accessing D-BUS",
		PUrl:        "pkg:rpm/rhel/dbus-libs@1%3A1.10.24-7.el7?arch=x86_64&distro=rhel-7.4&epoch=1",
		CPE:         "cpe:2.3:a:dbus-libs:dbus-libs:1:x86_64:*:*:*:*:1:*",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "gpg-pubkey detected")
}

func TestRedhat6Parser(t *testing.T) {
	mock, err := mock.New("./testdata/packages_redhat6.toml", nil)
	if err != nil {
		t.Fatal(err)
	}

	pf := &inventory.Platform{
		Name:    "redhat",
		Version: "6.2",
		Arch:    "x86_64",
		Family:  []string{"redhat", "linux", "unix", "os"},
		Labels: map[string]string{
			"distro-id": "rhel",
		},
	}

	c, err := mock.RunCommand("rpm -qa --queryformat '%{NAME} %{EPOCH}:%{VERSION}-%{RELEASE} %{ARCH} %{SUMMARY}\\n'")
	if err != nil {
		t.Fatal(err)
	}

	m := packages.ParseRpmPackages(pf, c.Stdout)
	assert.Equal(t, 8, len(m), "detected the right amount of packages")

	var p packages.Package
	p = packages.Package{
		Name:        "ElectricFence",
		Version:     "2.1-3",
		Arch:        "i386",
		Description: "A debugger which detects memory allocation violations.",
		PUrl:        "pkg:rpm/rhel/ElectricFence@2.1-3?arch=i386&distro=rhel-6.2",
		CPE:         "cpe:2.3:a:electricfence:electricfence:2.1-3:i386:*:*:*:*:*:*",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "ElectricFence")

	p = packages.Package{
		Name:        "shadow-utils",
		Version:     "1:19990827-10",
		Epoch:       "1",
		Arch:        "i386",
		Description: "Utilities for managing shadow password files and user/group accounts.",
		PUrl:        "pkg:rpm/rhel/shadow-utils@1%3A19990827-10?arch=i386&distro=rhel-6.2&epoch=1",
		CPE:         "cpe:2.3:a:shadow-utils:shadow-utils:1:i386:*:*:*:*:1:*",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "shadow-utils")

	p = packages.Package{
		Name:        "arpwatch",
		Version:     "1:2.1a4-19",
		Epoch:       "1",
		Arch:        "i386",
		Description: "Network monitoring tools for tracking IP addresses on a network.",
		PUrl:        "pkg:rpm/rhel/arpwatch@1%3A2.1a4-19?arch=i386&distro=rhel-6.2&epoch=1",
		CPE:         "cpe:2.3:a:arpwatch:arpwatch:1:i386:*:*:*:*:1:*",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "arpwatch")

	p = packages.Package{
		Name:        "bash",
		Version:     "1.14.7-22",
		Arch:        "i386",
		Description: "The GNU Bourne Again shell (bash) version 1.14.",
		PUrl:        "pkg:rpm/rhel/bash@1.14.7-22?arch=i386&distro=rhel-6.2",
		CPE:         "cpe:2.3:a:bash:bash:1.14.7-22:i386:*:*:*:*:*:*",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "bash")
}

func TestPhoton4ImageParser(t *testing.T) {
	// to create this test file, run the following command:
	// mondoo scan docker image photon:4.0 --record
	mock, err := mock.New("./testdata/packages_photon_image.toml", nil)
	if err != nil {
		t.Fatal(err)
	}

	rpmTmpDir, err := os.MkdirTemp(os.TempDir(), "mondoo-rpmdb")
	require.NoError(t, err)
	defer os.RemoveAll(rpmTmpDir)

	fWriter, err := os.Create(filepath.Join(rpmTmpDir, "rpmdb.sqlite"))
	require.NoError(t, err)
	defer fWriter.Close()

	f, err := mock.FileSystem().Open(filepath.Join("/var/lib/rpm", "rpmdb.sqlite"))
	require.NoError(t, err)
	defer f.Close()

	_, err = io.Copy(fWriter, f)
	require.NoError(t, err)

	packageList := bytes.Buffer{}
	db, err := rpmdb.Open(filepath.Join(rpmTmpDir, "rpmdb.sqlite"))
	require.NoError(t, err)

	pkgList, err := db.ListPackages()
	require.NoError(t, err)

	for _, pkg := range pkgList {
		packageList.WriteString(fmt.Sprintf("%s %d:%s-%s %s %s\n", pkg.Name, pkg.EpochNum(), pkg.Version, pkg.Release, pkg.Arch, pkg.Summary))
	}

	pf := &inventory.Platform{
		Name:    "photon",
		Version: "3.0",
		Arch:    "x86_64",
		Family:  []string{"linux", "unix", "os"},
		Labels: map[string]string{
			"distro-id": "photon",
		},
	}

	m := packages.ParseRpmPackages(pf, &packageList)
	assert.Equal(t, 36, len(m), "detected the right amount of packages")

	var p packages.Package
	p = packages.Package{
		Name:        "ncurses-libs",
		Version:     "6.2-6.ph4",
		Arch:        "x86_64",
		Description: "Ncurses Libraries",
		PUrl:        "pkg:rpm/photon/ncurses-libs@6.2-6.ph4?arch=x86_64&distro=photon-3.0",
		CPE:         "cpe:2.3:a:ncurses-libs:ncurses-libs:6.2-6.ph4:x86_64:*:*:*:*:*:*",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "ncurses-libs")

	p = packages.Package{
		Name:        "bash",
		Version:     "5.0-2.ph4",
		Arch:        "x86_64",
		Description: "Bourne-Again SHell",
		PUrl:        "pkg:rpm/photon/bash@5.0-2.ph4?arch=x86_64&distro=photon-3.0",
		CPE:         "cpe:2.3:a:bash:bash:5.0-2.ph4:x86_64:*:*:*:*:*:*",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "bash")

	p = packages.Package{
		Name:        "sqlite-libs",
		Version:     "3.38.5-1.ph4",
		Arch:        "x86_64",
		Description: "sqlite3 library",
		PUrl:        "pkg:rpm/photon/sqlite-libs@3.38.5-1.ph4?arch=x86_64&distro=photon-3.0",
		CPE:         "cpe:2.3:a:sqlite-libs:sqlite-libs:3.38.5-1.ph4:x86_64:*:*:*:*:*:*",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "sqlite-libs")
}
