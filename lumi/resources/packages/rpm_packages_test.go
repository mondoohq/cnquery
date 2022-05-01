package packages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/packages"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestRedhat7Parser(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/packages_redhat7.toml")
	if err != nil {
		t.Fatal(err)
	}

	c, err := mock.RunCommand("rpm -qa --queryformat '%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE} %{ARCH} %{SUMMARY}\\n'")
	if err != nil {
		t.Fatal(err)
	}

	m := packages.ParseRpmPackages(c.Stdout)
	assert.Equal(t, 144, len(m), "detected the right amount of packages")

	var p packages.Package
	p = packages.Package{
		Name:        "ncurses-base",
		Version:     "5.9-14.20130511.el7_4",
		Arch:        "noarch",
		Description: "Descriptions of common terminals",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "ncurses-base")

	p = packages.Package{
		Name:        "libstdc++",
		Version:     "4.8.5-28.el7_5.1",
		Arch:        "x86_64",
		Description: "GNU Standard C++ Library",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "libstdc detected")

	p = packages.Package{
		Name:        "iputils",
		Version:     "20160308-10.el7",
		Arch:        "x86_64",
		Description: "Network monitoring tools including ping",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "gpg-pubkey detected")

	p = packages.Package{
		Name:        "openssl-libs",
		Version:     "1:1.0.2k-12.el7",
		Arch:        "x86_64",
		Description: "A general purpose cryptography library with TLS implementation",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "gpg-pubkey detected")

	p = packages.Package{
		Name:        "dbus-libs",
		Version:     "1:1.10.24-7.el7",
		Arch:        "x86_64",
		Description: "Libraries for accessing D-BUS",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "gpg-pubkey detected")
}

func TestRedhat6Parser(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/packages_redhat6.toml")
	if err != nil {
		t.Fatal(err)
	}

	c, err := mock.RunCommand("rpm -qa --queryformat '%{NAME} %{EPOCH}:%{VERSION}-%{RELEASE} %{ARCH} %{SUMMARY}\\n'")
	if err != nil {
		t.Fatal(err)
	}

	m := packages.ParseRpmPackages(c.Stdout)
	assert.Equal(t, 8, len(m), "detected the right amount of packages")

	var p packages.Package
	p = packages.Package{
		Name:        "ElectricFence",
		Version:     "2.1-3",
		Arch:        "i386",
		Description: "A debugger which detects memory allocation violations.",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "ElectricFence")

	p = packages.Package{
		Name:        "shadow-utils",
		Version:     "1:19990827-10",
		Arch:        "i386",
		Description: "Utilities for managing shadow password files and user/group accounts.",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "shadow-utils")

	p = packages.Package{
		Name:        "arpwatch",
		Version:     "1:2.1a4-19",
		Arch:        "i386",
		Description: "Network monitoring tools for tracking IP addresses on a network.",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "arpwatch")

	p = packages.Package{
		Name:        "bash",
		Version:     "1.14.7-22",
		Arch:        "i386",
		Description: "The GNU Bourne Again shell (bash) version 1.14.",
		Format:      packages.RpmPkgFormat,
	}
	assert.Contains(t, m, p, "bash")
}
