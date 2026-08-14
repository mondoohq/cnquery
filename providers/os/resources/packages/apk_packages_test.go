// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
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

	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/packages_apk.toml"))
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
		Name:        "musl",
		Version:     "1510953106:1.1.18-r2",
		Epoch:       "1510953106",
		Arch:        "x86_64",
		Description: "the musl c library (libc) implementation",
		License:     "MIT",
		Origin:      "musl",
		PUrl:        "pkg:apk/alpine/musl@1510953106:1.1.18-r2?arch=x86_64&distro=alpine-3.7.0&epoch=1510953106",
		CPEs: []string{
			"cpe:2.3:a:*:musl:1.1.18-r2:*:*:*:*:*:x86_64:*",
		},
		Format:         AlpinePkgFormat,
		FilesAvailable: PkgFilesIncluded,
		Files: []FileRecord{
			{
				Path: "/lib/apk/db/installed",
			},
		},
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	p = Package{
		Name:        "libressl2.6-libcrypto",
		License:     "custom",
		Version:     "1510257703:2.6.3-r0",
		Epoch:       "1510257703",
		Arch:        "x86_64",
		Description: "libressl libcrypto library",
		Origin:      "libressl",
		PUrl:        "pkg:apk/alpine/libressl2.6-libcrypto@1510257703:2.6.3-r0?arch=x86_64&distro=alpine-3.7.0&epoch=1510257703",
		CPEs: []string{
			"cpe:2.3:a:*:libressl2.6-libcrypto:2.6.3-r0:*:*:*:*:*:x86_64:*",
		},
		Format:         AlpinePkgFormat,
		FilesAvailable: PkgFilesIncluded,
		Files: []FileRecord{
			{
				Path: "/lib/apk/db/installed",
			},
		},
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	p = Package{
		Name:        "libressl2.6-libssl",
		License:     "custom",
		Version:     "1510257703:2.6.3-r0",
		Epoch:       "1510257703",
		Arch:        "x86_64",
		Description: "libressl libssl library",
		Origin:      "libressl",
		PUrl:        "pkg:apk/alpine/libressl2.6-libssl@1510257703:2.6.3-r0?arch=x86_64&distro=alpine-3.7.0&epoch=1510257703",
		CPEs: []string{
			"cpe:2.3:a:*:libressl2.6-libssl:2.6.3-r0:*:*:*:*:*:x86_64:*",
		},
		Format:         AlpinePkgFormat,
		FilesAvailable: PkgFilesIncluded,
		Files: []FileRecord{
			{
				Path: "/lib/apk/db/installed",
			},
		},
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	p = Package{
		Name:        "apk-tools",
		License:     "GPL2",
		Version:     "1515485577:2.8.2-r0",
		Epoch:       "1515485577",
		Arch:        "x86_64",
		Description: "Alpine Package Keeper - package manager for alpine",
		Origin:      "apk-tools",
		PUrl:        "pkg:apk/alpine/apk-tools@1515485577:2.8.2-r0?arch=x86_64&distro=alpine-3.7.0&epoch=1515485577",
		CPEs: []string{
			"cpe:2.3:a:*:apk-tools:2.8.2-r0:*:*:*:*:*:x86_64:*",
		},
		Format:         AlpinePkgFormat,
		FilesAvailable: PkgFilesIncluded,
		Files: []FileRecord{
			{
				Path: "/lib/apk/db/installed",
			},
		},
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	p = Package{
		Name:        "busybox",
		License:     "GPL2",
		Version:     "1513075346:1.27.2-r7",
		Epoch:       "1513075346",
		Arch:        "x86_64",
		Description: "Size optimized toolbox of many common UNIX utilities",
		Origin:      "busybox",
		PUrl:        "pkg:apk/alpine/busybox@1513075346:1.27.2-r7?arch=x86_64&distro=alpine-3.7.0&epoch=1513075346",
		CPEs: []string{
			"cpe:2.3:a:*:busybox:1.27.2-r7:*:*:*:*:*:x86_64:*",
		},
		Format:         AlpinePkgFormat,
		FilesAvailable: PkgFilesIncluded,
		Files: []FileRecord{
			{
				Path: "/lib/apk/db/installed",
			},
		},
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)

	p = Package{
		Name:        "alpine-baselayout",
		License:     "GPL2",
		Version:     "1510075862:3.0.5-r2",
		Epoch:       "1510075862",
		Arch:        "x86_64",
		Description: "Alpine base dir structure and init scripts",
		Origin:      "alpine-baselayout",
		PUrl:        "pkg:apk/alpine/alpine-baselayout@1510075862:3.0.5-r2?arch=x86_64&distro=alpine-3.7.0&epoch=1510075862",
		CPEs: []string{
			"cpe:2.3:a:*:alpine-baselayout:3.0.5-r2:*:*:*:*:*:x86_64:*",
		},
		Format:         AlpinePkgFormat,
		FilesAvailable: PkgFilesIncluded,
		Files: []FileRecord{
			{Path: "/lib/apk/db/installed"},
		},
	}
	assert.Equal(t, p, findPkg(m, p.Name), p.Name)
}

func TestApkUpdateParser(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/updates_apk.toml"))
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

func TestApkField(t *testing.T) {
	// The parser used the regexp `^([A-Za-z]):(.*)$` before.
	tests := []struct {
		line  string
		key   byte
		value string
		ok    bool
	}{
		{"P:musl", 'P', "musl", true},
		{"V:1.2.5-r0", 'V', "1.2.5-r0", true},
		{"t:1234567890", 't', "1234567890", true},
		{"L:MIT", 'L', "MIT", true},
		// an empty value still matches, because the regexp used `.*`
		{"T:", 'T', "", true},
		// file lines are fields too, the parser just keeps none of them
		{"R:libc.musl-x86_64.so.1", 'R', "libc.musl-x86_64.so.1", true},
		// the key must be exactly one ASCII letter followed by a colon
		{"PP:musl", 0, "", false},
		{"1:musl", 0, "", false},
		{":musl", 0, "", false},
		{"P", 0, "", false},
		{"", 0, "", false},
		{" P:musl", 0, "", false},
	}
	for _, test := range tests {
		key, value, ok := apkField([]byte(test.line))
		assert.Equal(t, test.ok, ok, "match for %q", test.line)
		assert.Equal(t, test.key, key, "key for %q", test.line)
		assert.Equal(t, test.value, string(value), "value for %q", test.line)
	}
}

// apkBenchDB builds a database in the shape of /lib/apk/db/installed. Most
// lines of a real database list directories and files.
func apkBenchDB(pkgCount int, filesPerPkg int) []byte {
	var buf bytes.Buffer
	for i := 0; i < pkgCount; i++ {
		name := "package-" + strconv.Itoa(i)
		buf.WriteString("C:Q1eVpkasdfjkl" + strconv.Itoa(i) + "=\n")
		buf.WriteString("P:" + name + "\n")
		buf.WriteString("V:1." + strconv.Itoa(i) + ".0-r0\n")
		buf.WriteString("A:x86_64\n")
		buf.WriteString("S:" + strconv.Itoa(10000+i) + "\n")
		buf.WriteString("I:" + strconv.Itoa(40000+i) + "\n")
		buf.WriteString("T:short description for " + name + "\n")
		buf.WriteString("U:https://example.org/" + name + "\n")
		buf.WriteString("L:MIT\n")
		buf.WriteString("o:" + name + "\n")
		buf.WriteString("m:Alpine Developers <foo@example.org>\n")
		buf.WriteString("t:176000000" + strconv.Itoa(i%10) + "\n")
		buf.WriteString("c:abcdef0123456789abcdef0123456789abcdef01\n")
		buf.WriteString("D:so:libc.musl-x86_64.so.1 so:libz.so.1\n")
		buf.WriteString("p:cmd:" + name + "=1.0.0-r0\n")
		buf.WriteString("F:usr/lib/" + name + "\n")
		for j := 0; j < filesPerPkg; j++ {
			buf.WriteString("R:file-" + strconv.Itoa(j) + ".so\n")
			buf.WriteString("Z:Q1abcdefghijklmnop" + strconv.Itoa(j) + "=\n")
		}
		buf.WriteString("\n")
	}
	return buf.Bytes()
}

func BenchmarkParseApkDbPackages(b *testing.B) {
	data := apkBenchDB(60, 60)
	pf := &inventory.Platform{Name: "alpine", Version: "3.21", Arch: "x86_64"}

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pkgs := ParseApkDbPackages(pf, bytes.NewReader(data))
		if len(pkgs) != 60 {
			b.Fatalf("expected 60 packages, got %d", len(pkgs))
		}
	}
}
