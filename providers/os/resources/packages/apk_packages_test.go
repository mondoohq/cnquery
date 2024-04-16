// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
)

func TestAlpineApkdbParser(t *testing.T) {
	pf := &inventory.Platform{
		Name:    "alpine",
		Version: "3.7.0",
		Arch:    "x86_64",
		Family:  []string{"linux", "unix", "os"},
		Labels: map[string]string{
			"distro-id": "alpine",
		},
	}

	mock, err := mock.New(0, "./testdata/packages_apk.toml", &inventory.Asset{})
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.FileSystem().Open("/lib/apk/db/installed")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	m := ParseApkDbPackages(pf, f)
	assert.Equal(t, 7, len(m), "detected the right amount of packages")

	p := Package{
		Name:           "musl",
		Version:        "1510953106:1.1.18-r2",
		Epoch:          "1510953106",
		Arch:           "x86_64",
		Description:    "the musl c library (libc) implementation",
		Origin:         "musl",
		PUrl:           "pkg:apk/alpine/musl@1510953106%3A1.1.18-r2?arch=x86_64&distro=alpine-3.7.0&epoch=1510953106",
		CPE:            "cpe:2.3:a:musl:musl:1510953106:x86_64:*:*:*:*:x86_64:*",
		Format:         AlpinePkgFormat,
		FilesAvailable: PkgFilesIncluded,
		Files: []FileRecord{
			{
				Path: "lib/libc.musl-x86_64.so.1",
			},
			{
				Path: "lib/ld-musl-x86_64.so.1",
			},
		},
	}
	assert.Equal(t, findPkg(m, p.Name), p, p.Name)

	p = Package{
		Name:           "libressl2.6-libcrypto",
		Version:        "1510257703:2.6.3-r0",
		Epoch:          "1510257703",
		Arch:           "x86_64",
		Description:    "libressl libcrypto library",
		Origin:         "libressl",
		PUrl:           "pkg:apk/alpine/libressl2.6-libcrypto@1510257703%3A2.6.3-r0?arch=x86_64&distro=alpine-3.7.0&epoch=1510257703",
		CPE:            "cpe:2.3:a:libressl2.6-libcrypto:libressl2.6-libcrypto:1510257703:x86_64:*:*:*:*:x86_64:*",
		Format:         AlpinePkgFormat,
		FilesAvailable: PkgFilesIncluded,
		Files: []FileRecord{
			{
				Path: "etc/ssl/cert.pem",
			},
			{
				Path: "etc/ssl/x509v3.cnf",
			},
			{
				Path: "etc/ssl/openssl.cnf",
			},
			{
				Path: "lib/libcrypto.so.42",
			},
			{
				Path: "lib/libcrypto.so.42.0.0",
			},
			{
				Path: "usr/lib/libcrypto.so.42",
			},
			{
				Path: "usr/lib/libcrypto.so.42.0.0",
			},
		},
	}
	assert.Equal(t, findPkg(m, p.Name), p, p.Name)

	p = Package{
		Name:           "libressl2.6-libssl",
		Version:        "1510257703:2.6.3-r0",
		Epoch:          "1510257703",
		Arch:           "x86_64",
		Description:    "libressl libssl library",
		Origin:         "libressl",
		PUrl:           "pkg:apk/alpine/libressl2.6-libssl@1510257703%3A2.6.3-r0?arch=x86_64&distro=alpine-3.7.0&epoch=1510257703",
		CPE:            "cpe:2.3:a:libressl2.6-libssl:libressl2.6-libssl:1510257703:x86_64:*:*:*:*:x86_64:*",
		Format:         AlpinePkgFormat,
		FilesAvailable: PkgFilesIncluded,
		Files: []FileRecord{
			{
				Path: "lib/libssl.so.44.0.1",
			},
			{
				Path: "lib/libssl.so.44",
			},
			{
				Path: "usr/lib/libssl.so.44.0.1",
			},
			{
				Path: "usr/lib/libssl.so.44",
			},
		},
	}
	assert.Equal(t, findPkg(m, p.Name), p, p.Name)

	p = Package{
		Name:           "apk-tools",
		Version:        "1515485577:2.8.2-r0",
		Epoch:          "1515485577",
		Arch:           "x86_64",
		Description:    "Alpine Package Keeper - package manager for alpine",
		Origin:         "apk-tools",
		PUrl:           "pkg:apk/alpine/apk-tools@1515485577%3A2.8.2-r0?arch=x86_64&distro=alpine-3.7.0&epoch=1515485577",
		CPE:            "cpe:2.3:a:apk-tools:apk-tools:1515485577:x86_64:*:*:*:*:x86_64:*",
		Format:         AlpinePkgFormat,
		FilesAvailable: PkgFilesIncluded,
		Files: []FileRecord{
			{
				Path: "sbin/apk",
			},
		},
	}
	assert.Equal(t, findPkg(m, p.Name), p, p.Name)

	p = Package{
		Name:           "busybox",
		Version:        "1513075346:1.27.2-r7",
		Epoch:          "1513075346",
		Arch:           "x86_64",
		Description:    "Size optimized toolbox of many common UNIX utilities",
		Origin:         "busybox",
		PUrl:           "pkg:apk/alpine/busybox@1513075346%3A1.27.2-r7?arch=x86_64&distro=alpine-3.7.0&epoch=1513075346",
		CPE:            "cpe:2.3:a:busybox:busybox:1513075346:x86_64:*:*:*:*:x86_64:*",
		Format:         AlpinePkgFormat,
		FilesAvailable: PkgFilesIncluded,
		Files: []FileRecord{
			{
				Path: "bin/busybox",
			},
			{
				Path: "bin/sh",
			},
			{
				Path: "etc/securetty",
			},
			{
				Path: "etc/udhcpd.conf",
			},
			{
				Path: "etc/logrotate.d/acpid",
			},
			{
				Path: "etc/network/if-up.d/dad",
			},
		},
	}
	assert.Equal(t, findPkg(m, p.Name), p, p.Name)

	p = Package{
		Name:           "alpine-baselayout",
		Version:        "1510075862:3.0.5-r2",
		Epoch:          "1510075862",
		Arch:           "x86_64",
		Description:    "Alpine base dir structure and init scripts",
		Origin:         "alpine-baselayout",
		PUrl:           "pkg:apk/alpine/alpine-baselayout@1510075862%3A3.0.5-r2?arch=x86_64&distro=alpine-3.7.0&epoch=1510075862",
		CPE:            "cpe:2.3:a:alpine-baselayout:alpine-baselayout:1510075862:x86_64:*:*:*:*:x86_64:*",
		Format:         AlpinePkgFormat,
		FilesAvailable: PkgFilesIncluded,
		Files: []FileRecord{
			{Path: "etc/hosts"},
			{Path: "etc/sysctl.conf"},
			{Path: "etc/group"},
			{Path: "etc/protocols"},
			{Path: "etc/fstab"},
			{Path: "etc/mtab"},
			{Path: "etc/profile"},
			{Path: "etc/TZ"},
			{Path: "etc/shells"},
			{Path: "etc/motd"},
			{Path: "etc/inittab"},
			{Path: "etc/hostname"},
			{Path: "etc/modules"},
			{Path: "etc/services"},
			{Path: "etc/shadow"},
			{Path: "etc/passwd"},
			{Path: "etc/profile.d/color_prompt"},
			{Path: "etc/sysctl.d/00-alpine.conf"},
			{Path: "etc/modprobe.d/i386.conf"},
			{Path: "etc/modprobe.d/blacklist.conf"},
			{Path: "etc/modprobe.d/aliases.conf"},
			{Path: "etc/modprobe.d/kms.conf"},
			{Path: "etc/crontabs/root"},
			{Path: "sbin/mkmntdirs"},
			{Path: "var/run"},
			{Path: "var/spool/cron/crontabs"},
		},
	}
	assert.Equal(t, findPkg(m, p.Name), p, p.Name)
}

func TestApkUpdateParser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/updates_apk.toml", &inventory.Asset{})
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

	update := m["busybox"]
	assert.Equal(t, "busybox", update.Name, "pkg name detected")
	assert.Equal(t, "1.28.4-r0", update.Version, "pkg version detected")
	assert.Equal(t, "1.28.4-r1", update.Available, "pkg available version detected")

	update = m["ssl_client"]
	assert.Equal(t, "ssl_client", update.Name, "pkg name detected")
	assert.Equal(t, "1.28.4-r0", update.Version, "pkg version detected")
	assert.Equal(t, "1.28.4-r1", update.Available, "pkg available version detected")
}
