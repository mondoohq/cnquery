// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bytes"
	"fmt"
	"testing"

	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
)

func TestRedhat8Parser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/packages_redhat8.toml", &inventory.Asset{})
	if err != nil {
		t.Fatal(err)
	}

	pf := &inventory.Platform{
		Name:    "redhat",
		Version: "8.4",
		Arch:    "x86_64",
		Family:  []string{"redhat", "linux", "unix", "os"},
		Labels: map[string]string{
			"distro-id": "rhel",
		},
	}

	c, err := mock.RunCommand("rpm -qa --queryformat '%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE} %{ARCH}__%{VENDOR}__%{SUMMARY}\\n'")
	if err != nil {
		t.Fatal(err)
	}

	m := ParseRpmPackages(pf, c.Stdout)
	assert.Equal(t, 190, len(m), "detected the right amount of packages")

	p := Package{
		Name:        "ncurses-base",
		Version:     "6.1-7.20180224.el8",
		Vendor:      "Red Hat, Inc.",
		Arch:        "noarch",
		Description: "Descriptions of common terminals",
		PUrl:        "pkg:rpm/rhel/ncurses-base@6.1-7.20180224.el8?arch=noarch&distro=rhel-8.4",
		CPEs: []string{
			"cpe:2.3:a:red_hat\\,_inc.:ncurses-base:6.1-7.20180224.el8:*:*:*:*:*:noarch:*",
			"cpe:2.3:a:red_hat\\,_inc.:ncurses-base:6.1-7.20180224:*:*:*:*:*:noarch:*",
			"cpe:2.3:a:red_hat\\,_inc.:ncurses-base:6.1:*:*:*:*:*:noarch:*",
			"cpe:2.3:a:red_hat\\,_inc.:ncurses-base:6.1-7.20180224.el8:*:*:*:*:*:*:*",
			"cpe:2.3:a:red_hat\\,_inc.:ncurses-base:6.1-7.20180224:*:*:*:*:*:*:*",
			"cpe:2.3:a:red_hat\\,_inc.:ncurses-base:6.1:*:*:*:*:*:*:*",
		},
		Format:         RpmPkgFormat,
		FilesAvailable: PkgFilesAsync,
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	p = Package{
		Name:        "libstdc++",
		Version:     "8.4.1-1.el8",
		Vendor:      "Red Hat, Inc.",
		Arch:        "x86_64",
		Description: "GNU Standard C++ Library",
		PUrl:        "pkg:rpm/rhel/libstdc%2B%2B@8.4.1-1.el8?arch=x86_64&distro=rhel-8.4",
		CPEs: []string{
			"cpe:2.3:a:red_hat\\,_inc.:libstdc\\+\\+:8.4.1-1.el8:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:libstdc\\+\\+:8.4.1-1:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:libstdc\\+\\+:8.4.1:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:libstdc\\+\\+:8.4.1-1.el8:*:*:*:*:*:*:*",
			"cpe:2.3:a:red_hat\\,_inc.:libstdc\\+\\+:8.4.1-1:*:*:*:*:*:*:*",
			"cpe:2.3:a:red_hat\\,_inc.:libstdc\\+\\+:8.4.1:*:*:*:*:*:*:*",
		},
		Format:         RpmPkgFormat,
		FilesAvailable: PkgFilesAsync,
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	p = Package{
		Name:        "iptables-libs",
		Version:     "1.8.4-17.el8_4.1",
		Vendor:      "Red Hat, Inc.",
		Arch:        "x86_64",
		Description: "iptables libraries",
		PUrl:        "pkg:rpm/rhel/iptables-libs@1.8.4-17.el8_4.1?arch=x86_64&distro=rhel-8.4",
		CPEs: []string{
			"cpe:2.3:a:red_hat\\,_inc.:iptables-libs:1.8.4-17.el8_4.1:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:iptables-libs:1.8.4-17:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:iptables-libs:1.8.4:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:iptables-libs:1.8.4-17.el8_4.1:*:*:*:*:*:*:*",
			"cpe:2.3:a:red_hat\\,_inc.:iptables-libs:1.8.4-17:*:*:*:*:*:*:*",
			"cpe:2.3:a:red_hat\\,_inc.:iptables-libs:1.8.4:*:*:*:*:*:*:*",
		},
		Format:         RpmPkgFormat,
		FilesAvailable: PkgFilesAsync,
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	p = Package{
		Name:        "openssl-libs",
		Version:     "1:1.1.1g-15.el8_3",
		Vendor:      "Red Hat, Inc.",
		Epoch:       "1",
		Arch:        "x86_64",
		Description: "A general purpose cryptography library with TLS implementation",
		PUrl:        "pkg:rpm/rhel/openssl-libs@1%3A1.1.1g-15.el8_3?arch=x86_64&distro=rhel-8.4&epoch=1",
		CPEs: []string{
			"cpe:2.3:a:red_hat\\,_inc.:openssl-libs:1.1.1g-15.el8_3:1:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:openssl-libs:1.1.1g-15:1:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:openssl-libs:1.1.1g:1:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:openssl-libs:1.1.1g-15.el8_3:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:openssl-libs:1.1.1g-15:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:openssl-libs:1.1.1g:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:openssl-libs:1.1.1g-15.el8_3:*:*:*:*:*:*:*",
			"cpe:2.3:a:red_hat\\,_inc.:openssl-libs:1.1.1g-15:*:*:*:*:*:*:*",
			"cpe:2.3:a:red_hat\\,_inc.:openssl-libs:1.1.1g:*:*:*:*:*:*:*",
		},
		Format:         RpmPkgFormat,
		FilesAvailable: PkgFilesAsync,
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	p = Package{
		Name:        "dbus-libs",
		Version:     "1:1.12.8-12.el8_4.2",
		Vendor:      "Red Hat, Inc.",
		Epoch:       "1",
		Arch:        "x86_64",
		Description: "Libraries for accessing D-BUS",
		PUrl:        "pkg:rpm/rhel/dbus-libs@1%3A1.12.8-12.el8_4.2?arch=x86_64&distro=rhel-8.4&epoch=1",
		CPEs: []string{
			"cpe:2.3:a:red_hat\\,_inc.:dbus-libs:1.12.8-12.el8_4.2:1:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:dbus-libs:1.12.8-12:1:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:dbus-libs:1.12.8:1:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:dbus-libs:1.12.8-12.el8_4.2:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:dbus-libs:1.12.8-12:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:dbus-libs:1.12.8:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:dbus-libs:1.12.8-12.el8_4.2:*:*:*:*:*:*:*",
			"cpe:2.3:a:red_hat\\,_inc.:dbus-libs:1.12.8-12:*:*:*:*:*:*:*",
			"cpe:2.3:a:red_hat\\,_inc.:dbus-libs:1.12.8:*:*:*:*:*:*:*",
		},
		Format:         RpmPkgFormat,
		FilesAvailable: PkgFilesAsync,
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	// fetch package files
	p = Package{
		Name:        "which",
		Version:     "2.21-12.el8",
		Vendor:      "Red Hat, Inc.",
		Epoch:       "",
		Arch:        "x86_64",
		Description: "Displays where a particular program in your path is located",
		PUrl:        "pkg:rpm/rhel/which@2.21-12.el8?arch=x86_64&distro=rhel-8.4",
		CPEs: []string{
			"cpe:2.3:a:red_hat\\,_inc.:which:2.21-12.el8:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:which:2.21:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:red_hat\\,_inc.:which:2.21-12.el8:*:*:*:*:*:*:*",
			"cpe:2.3:a:red_hat\\,_inc.:which:2.21:*:*:*:*:*:*:*",
		},
		Format:         RpmPkgFormat,
		FilesAvailable: PkgFilesAsync,
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	mgr := &RpmPkgManager{
		conn:     mock,
		platform: pf,
	}
	pkgFiles, err := mgr.Files(p.Name, p.Version, p.Arch)
	require.NoError(t, err)
	assert.Equal(t, 15, len(pkgFiles), "detected the right amount of package files")
	assert.Contains(t, pkgFiles, FileRecord{Path: "/usr/share/doc/which"})
	assert.Contains(t, pkgFiles, FileRecord{Path: "/usr/share/info/which.info.gz"})
}

func TestPhoton4ImageParser(t *testing.T) {
	// To get this data, run the following command on a Photon 4 system:
	// tdnf info ncurses-libs bash sqlite-libs

	epoch := int(0)
	pkgList := []*rpmdb.PackageInfo{
		{
			Name:    "ncurses-libs",
			Epoch:   &epoch,
			Version: "6.2",
			Release: "6.ph4",
			Arch:    "x86_64",
			Vendor:  "VMware, Inc.",
			Summary: "Ncurses Libraries",
		},
		{
			Name:    "bash",
			Epoch:   &epoch,
			Version: "5.0",
			Release: "4.ph4",
			Arch:    "x86_64",
			Vendor:  "VMware, Inc.",
			Summary: "Bourne-Again SHell",
		},
		{
			Name:    "sqlite-libs",
			Epoch:   &epoch,
			Version: "3.38.5",
			Release: "4.ph4",
			Arch:    "x86_64",
			Vendor:  "VMware, Inc.",
			Summary: "sqlite3 library",
		},
	}

	var packageList bytes.Buffer
	for _, pkg := range pkgList {
		packageList.WriteString(fmt.Sprintf("%s %d:%s-%s %s__%s__%s\n", pkg.Name, pkg.EpochNum(), pkg.Version, pkg.Release, pkg.Arch, pkg.Vendor, pkg.Summary))
	}

	pf := &inventory.Platform{
		Name:    "photon",
		Version: "4.0",
		Arch:    "x86_64",
		Family:  []string{"linux", "unix", "os"},
		Labels: map[string]string{
			"distro-id": "photon",
		},
	}

	m := ParseRpmPackages(pf, &packageList)
	assert.Equal(t, 3, len(m), "detected the right amount of packages")

	p := Package{
		Name:        "ncurses-libs",
		Version:     "6.2-6.ph4",
		Vendor:      "VMware, Inc.",
		Arch:        "x86_64",
		Description: "Ncurses Libraries",
		PUrl:        "pkg:rpm/photon/ncurses-libs@6.2-6.ph4?arch=x86_64&distro=photon-4.0",
		CPEs: []string{
			"cpe:2.3:a:vmware\\,_inc.:ncurses-libs:6.2-6.ph4:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:vmware\\,_inc.:ncurses-libs:6.2:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:vmware\\,_inc.:ncurses-libs:6.2-6.ph4:*:*:*:*:*:*:*",
			"cpe:2.3:a:vmware\\,_inc.:ncurses-libs:6.2:*:*:*:*:*:*:*",
		},
		Format:         RpmPkgFormat,
		FilesAvailable: PkgFilesAsync,
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	p = Package{
		Name:        "bash",
		Version:     "5.0-4.ph4",
		Vendor:      "VMware, Inc.",
		Arch:        "x86_64",
		Description: "Bourne-Again SHell",
		PUrl:        "pkg:rpm/photon/bash@5.0-4.ph4?arch=x86_64&distro=photon-4.0",
		CPEs: []string{
			"cpe:2.3:a:vmware\\,_inc.:bash:5.0-4.ph4:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:vmware\\,_inc.:bash:5.0:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:vmware\\,_inc.:bash:5.0-4.ph4:*:*:*:*:*:*:*",
			"cpe:2.3:a:vmware\\,_inc.:bash:5.0:*:*:*:*:*:*:*",
		},
		Format:         RpmPkgFormat,
		FilesAvailable: PkgFilesAsync,
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	p = Package{
		Name:        "sqlite-libs",
		Version:     "3.38.5-4.ph4",
		Vendor:      "VMware, Inc.",
		Arch:        "x86_64",
		Description: "sqlite3 library",
		PUrl:        "pkg:rpm/photon/sqlite-libs@3.38.5-4.ph4?arch=x86_64&distro=photon-4.0",
		CPEs: []string{
			"cpe:2.3:a:vmware\\,_inc.:sqlite-libs:3.38.5-4.ph4:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:vmware\\,_inc.:sqlite-libs:3.38.5-4:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:vmware\\,_inc.:sqlite-libs:3.38.5:*:*:*:*:*:x86_64:*",
			"cpe:2.3:a:vmware\\,_inc.:sqlite-libs:3.38.5-4.ph4:*:*:*:*:*:*:*",
			"cpe:2.3:a:vmware\\,_inc.:sqlite-libs:3.38.5-4:*:*:*:*:*:*:*",
			"cpe:2.3:a:vmware\\,_inc.:sqlite-libs:3.38.5:*:*:*:*:*:*:*",
		},
		Format:         RpmPkgFormat,
		FilesAvailable: PkgFilesAsync,
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)
}
