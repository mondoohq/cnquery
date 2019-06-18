package packages

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestPacmanParser(t *testing.T) {
	packages := `qpdfview 0.4.17beta1-4.1
usbmuxd 1.1.0+28+g46bdf3e-1
vertex-maia-themes 20171114-1
xfce4-power-manager 1.6.0.41.g9daecb5-1
xfce4-pulseaudio-plugin 0.3.2.r13.g553691a-1
zita-alsa-pcmi 0.2.0-3
zlib 1:1.2.11-2
zziplib 0.13.67-1`

	m := ParsePacmanPackages(strings.NewReader(packages))

	assert.Equal(t, 8, len(m), "detected the right amount of packages")
	var p Package
	p = Package{
		Name:    "qpdfview",
		Version: "0.4.17beta1-4.1",
	}
	assert.Contains(t, m, p, "pkg detected")

	p = Package{
		Name:    "vertex-maia-themes",
		Version: "20171114-1",
	}
	assert.Contains(t, m, p, "pkg detected")

	p = Package{
		Name:    "xfce4-pulseaudio-plugin",
		Version: "0.3.2.r13.g553691a-1",
	}
	assert.Contains(t, m, p, "pkg detected")
}

func TestAlpineApkdbParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "packages_apk.toml"})
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

func TestDpkgParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "packages_dpkg.toml"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.File("/var/lib/dpkg/status")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	m, err := ParseDpkgPackages(f)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 10, len(m), "detected the right amount of packages")

	var p Package
	p = Package{
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
	}
	assert.Contains(t, m, p, "fdisk detected")

	p = Package{
		Name:    "libaudit1",
		Version: "1:2.4-1+b1",
		Arch:    "amd64",
		Status:  "install ok installed",
		Origin:  "audit",
		Description: `Dynamic library for security auditing
The audit-libs package contains the dynamic libraries needed for
applications to use the audit framework. It is used to monitor systems for
security related events.`,
	}
	assert.Contains(t, m, p, "libaudit1 detected")
}

func TestMacOsXPackageParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "packages_macos.toml"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("system_profiler SPApplicationsDataType -xml")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := ParseMacOSPackages(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(m), "detected the right amount of packages")

	assert.Equal(t, "Preview", m[0].Name, "pkg name detected")
	assert.Equal(t, "10.0", m[0].Version, "pkg version detected")

	assert.Equal(t, "Contacts", m[1].Name, "pkg name detected")
	assert.Equal(t, "11.0", m[1].Version, "pkg version detected")
}

func TestRedhat7Parser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "packages_redhat7.toml"})
	if err != nil {
		t.Fatal(err)
	}

	c, err := mock.RunCommand("rpm -qa --queryformat '%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE} %{ARCH} %{SUMMARY}\\n'")
	if err != nil {
		t.Fatal(err)
	}

	m := ParseRpmPackages(c.Stdout)
	assert.Equal(t, 144, len(m), "detected the right amount of packages")

	var p Package
	p = Package{
		Name:        "ncurses-base",
		Version:     "5.9-14.20130511.el7_4",
		Arch:        "noarch",
		Description: "Descriptions of common terminals",
	}
	assert.Contains(t, m, p, "ncurses-base")

	p = Package{
		Name:        "libstdc++",
		Version:     "4.8.5-28.el7_5.1",
		Arch:        "x86_64",
		Description: "GNU Standard C++ Library",
	}
	assert.Contains(t, m, p, "libstdc detected")

	p = Package{
		Name:        "iputils",
		Version:     "20160308-10.el7",
		Arch:        "x86_64",
		Description: "Network monitoring tools including ping",
	}
	assert.Contains(t, m, p, "gpg-pubkey detected")

	p = Package{
		Name:        "openssl-libs",
		Version:     "1:1.0.2k-12.el7",
		Arch:        "x86_64",
		Description: "A general purpose cryptography library with TLS implementation",
	}
	assert.Contains(t, m, p, "gpg-pubkey detected")

	p = Package{
		Name:        "dbus-libs",
		Version:     "1:1.10.24-7.el7",
		Arch:        "x86_64",
		Description: "Libraries for accessing D-BUS",
	}
	assert.Contains(t, m, p, "gpg-pubkey detected")
}

func TestRedhat6Parser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "packages_redhat6.toml"})
	if err != nil {
		t.Fatal(err)
	}

	c, err := mock.RunCommand("rpm -qa --queryformat '%{NAME} %{EPOCH}:%{VERSION}-%{RELEASE} %{ARCH} %{SUMMARY}\\n'")
	if err != nil {
		t.Fatal(err)
	}

	m := ParseRpmPackages(c.Stdout)
	assert.Equal(t, 8, len(m), "detected the right amount of packages")

	var p Package
	p = Package{
		Name:        "ElectricFence",
		Version:     "2.1-3",
		Arch:        "i386",
		Description: "A debugger which detects memory allocation violations.",
	}
	assert.Contains(t, m, p, "ElectricFence")

	p = Package{
		Name:        "shadow-utils",
		Version:     "1:19990827-10",
		Arch:        "i386",
		Description: "Utilities for managing shadow password files and user/group accounts.",
	}
	assert.Contains(t, m, p, "shadow-utils")

	p = Package{
		Name:        "arpwatch",
		Version:     "1:2.1a4-19",
		Arch:        "i386",
		Description: "Network monitoring tools for tracking IP addresses on a network.",
	}
	assert.Contains(t, m, p, "arpwatch")

	p = Package{
		Name:        "bash",
		Version:     "1.14.7-22",
		Arch:        "i386",
		Description: "The GNU Bourne Again shell (bash) version 1.14.",
	}
	assert.Contains(t, m, p, "bash")
}
