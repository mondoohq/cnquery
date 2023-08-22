// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/providers/os/connection/mock"
	"go.mondoo.com/cnquery/providers/os/resources/packages"
)

func TestAlpineApkdbParser(t *testing.T) {
	mock, err := mock.New("./testdata/packages_apk.toml", nil)
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.FileSystem().Open("/lib/apk/db/installed")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	m := packages.ParseApkDbPackages(f)
	assert.Equal(t, 7, len(m), "detected the right amount of packages")

	var p packages.Package
	p = packages.Package{
		Name:        "musl",
		Version:     "1510953106:1.1.18-r2",
		Arch:        "x86_64",
		Description: "the musl c library (libc) implementation",
		Origin:      "musl",
		Format:      packages.AlpinePkgFormat,
	}
	assert.Contains(t, m, p, "musl detected")

	p = packages.Package{
		Name:        "libressl2.6-libcrypto",
		Version:     "1510257703:2.6.3-r0",
		Arch:        "x86_64",
		Description: "libressl libcrypto library",
		Origin:      "libressl",
		Format:      packages.AlpinePkgFormat,
	}
	assert.Contains(t, m, p, "libcrypto detected")

	p = packages.Package{
		Name:        "libressl2.6-libssl",
		Version:     "1510257703:2.6.3-r0",
		Arch:        "x86_64",
		Description: "libressl libssl library",
		Origin:      "libressl",
		Format:      packages.AlpinePkgFormat,
	}
	assert.Contains(t, m, p, "libssl detected")

	p = packages.Package{
		Name:        "apk-tools",
		Version:     "1515485577:2.8.2-r0",
		Arch:        "x86_64",
		Description: "Alpine Package Keeper - package manager for alpine",
		Origin:      "apk-tools",
		Format:      packages.AlpinePkgFormat,
	}
	assert.Contains(t, m, p, "apk-tools detected")

	p = packages.Package{
		Name:        "busybox",
		Version:     "1513075346:1.27.2-r7",
		Arch:        "x86_64",
		Description: "Size optimized toolbox of many common UNIX utilities",
		Origin:      "busybox",
		Format:      packages.AlpinePkgFormat,
	}
	assert.Contains(t, m, p, "apk-tools detected")

	p = packages.Package{
		Name:        "alpine-baselayout",
		Version:     "1510075862:3.0.5-r2",
		Arch:        "x86_64",
		Description: "Alpine base dir structure and init scripts",
		Origin:      "alpine-baselayout",
		Format:      packages.AlpinePkgFormat,
	}
	assert.Contains(t, m, p, "apk-tools detected")
}

func TestApkUpdateParser(t *testing.T) {
	mock, err := mock.New("./testdata/updates_apk.toml", nil)
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
