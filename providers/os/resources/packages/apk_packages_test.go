// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages_test

import (
	"testing"

	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/resources/packages"
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

	mock, err := mock.New(0, "./testdata/packages_apk.toml", nil)
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.FileSystem().Open("/lib/apk/db/installed")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	m := packages.ParseApkDbPackages(pf, f)
	assert.Equal(t, 7, len(m), "detected the right amount of packages")

	var p packages.Package
	p = packages.Package{
		Name:        "musl",
		Version:     "1510953106:1.1.18-r2",
		Epoch:       "1510953106",
		Arch:        "x86_64",
		Description: "the musl c library (libc) implementation",
		Origin:      "musl",
		PUrl:        "pkg:apk/alpine/musl@1510953106%3A1.1.18-r2?arch=x86_64&distro=alpine-3.7.0&epoch=1510953106",
		CPE:         "cpe:2.3:a:musl:musl:1510953106:x86_64:*:*:*:*:x86_64:*",
		Format:      packages.AlpinePkgFormat,
	}
	assert.Contains(t, m, p, "musl detected")

	p = packages.Package{
		Name:        "libressl2.6-libcrypto",
		Version:     "1510257703:2.6.3-r0",
		Epoch:       "1510257703",
		Arch:        "x86_64",
		Description: "libressl libcrypto library",
		Origin:      "libressl",
		PUrl:        "pkg:apk/alpine/libressl2.6-libcrypto@1510257703%3A2.6.3-r0?arch=x86_64&distro=alpine-3.7.0&epoch=1510257703",
		CPE:         "cpe:2.3:a:libressl2.6-libcrypto:libressl2.6-libcrypto:1510257703:x86_64:*:*:*:*:x86_64:*",
		Format:      packages.AlpinePkgFormat,
	}
	assert.Contains(t, m, p, "libcrypto detected")

	p = packages.Package{
		Name:        "libressl2.6-libssl",
		Version:     "1510257703:2.6.3-r0",
		Epoch:       "1510257703",
		Arch:        "x86_64",
		Description: "libressl libssl library",
		Origin:      "libressl",
		PUrl:        "pkg:apk/alpine/libressl2.6-libssl@1510257703%3A2.6.3-r0?arch=x86_64&distro=alpine-3.7.0&epoch=1510257703",
		CPE:         "cpe:2.3:a:libressl2.6-libssl:libressl2.6-libssl:1510257703:x86_64:*:*:*:*:x86_64:*",
		Format:      packages.AlpinePkgFormat,
	}
	assert.Contains(t, m, p, "libssl detected")

	p = packages.Package{
		Name:        "apk-tools",
		Version:     "1515485577:2.8.2-r0",
		Epoch:       "1515485577",
		Arch:        "x86_64",
		Description: "Alpine Package Keeper - package manager for alpine",
		Origin:      "apk-tools",
		PUrl:        "pkg:apk/alpine/apk-tools@1515485577%3A2.8.2-r0?arch=x86_64&distro=alpine-3.7.0&epoch=1515485577",
		CPE:         "cpe:2.3:a:apk-tools:apk-tools:1515485577:x86_64:*:*:*:*:x86_64:*",
		Format:      packages.AlpinePkgFormat,
	}
	assert.Contains(t, m, p, "apk-tools detected")

	p = packages.Package{
		Name:        "busybox",
		Version:     "1513075346:1.27.2-r7",
		Epoch:       "1513075346",
		Arch:        "x86_64",
		Description: "Size optimized toolbox of many common UNIX utilities",
		Origin:      "busybox",
		PUrl:        "pkg:apk/alpine/busybox@1513075346%3A1.27.2-r7?arch=x86_64&distro=alpine-3.7.0&epoch=1513075346",
		CPE:         "cpe:2.3:a:busybox:busybox:1513075346:x86_64:*:*:*:*:x86_64:*",
		Format:      packages.AlpinePkgFormat,
	}
	assert.Contains(t, m, p, "busybox detected")

	p = packages.Package{
		Name:        "alpine-baselayout",
		Version:     "1510075862:3.0.5-r2",
		Epoch:       "1510075862",
		Arch:        "x86_64",
		Description: "Alpine base dir structure and init scripts",
		Origin:      "alpine-baselayout",
		PUrl:        "pkg:apk/alpine/alpine-baselayout@1510075862%3A3.0.5-r2?arch=x86_64&distro=alpine-3.7.0&epoch=1510075862",
		CPE:         "cpe:2.3:a:alpine-baselayout:alpine-baselayout:1510075862:x86_64:*:*:*:*:x86_64:*",
		Format:      packages.AlpinePkgFormat,
	}
	assert.Contains(t, m, p, "alpine-baselayout detected")
}

func TestApkUpdateParser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/updates_apk.toml", nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("apk version -v -l '<'")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := packages.ParseApkUpdates(c.Stdout)
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
