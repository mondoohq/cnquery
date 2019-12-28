package packages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestAlpineApkdbParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/packages_apk.toml"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.File("/lib/apk/db/installed")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	m := ParseApkDbPackages(f)
	assert.Equal(t, 7, len(m), "detected the right amount of packages")

	var p Package
	p = Package{
		Name:        "musl",
		Version:     "1510953106:1.1.18-r2",
		Arch:        "x86_64",
		Description: "the musl c library (libc) implementation",
		Origin:      "musl",
	}
	assert.Contains(t, m, p, "musl detected")

	p = Package{
		Name:        "libressl2.6-libcrypto",
		Version:     "1510257703:2.6.3-r0",
		Arch:        "x86_64",
		Description: "libressl libcrypto library",
		Origin:      "libressl",
	}
	assert.Contains(t, m, p, "libcrypto detected")

	p = Package{
		Name:        "libressl2.6-libssl",
		Version:     "1510257703:2.6.3-r0",
		Arch:        "x86_64",
		Description: "libressl libssl library",
		Origin:      "libressl",
	}
	assert.Contains(t, m, p, "libssl detected")

	p = Package{
		Name:        "apk-tools",
		Version:     "1515485577:2.8.2-r0",
		Arch:        "x86_64",
		Description: "Alpine Package Keeper - package manager for alpine",
		Origin:      "apk-tools",
	}
	assert.Contains(t, m, p, "apk-tools detected")

	p = Package{
		Name:        "busybox",
		Version:     "1513075346:1.27.2-r7",
		Arch:        "x86_64",
		Description: "Size optimized toolbox of many common UNIX utilities",
		Origin:      "busybox",
	}
	assert.Contains(t, m, p, "apk-tools detected")

	p = Package{
		Name:        "alpine-baselayout",
		Version:     "1510075862:3.0.5-r2",
		Arch:        "x86_64",
		Description: "Alpine base dir structure and init scripts",
		Origin:      "alpine-baselayout",
	}
	assert.Contains(t, m, p, "apk-tools detected")
}
