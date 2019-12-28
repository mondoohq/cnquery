package packages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestRedhat7Parser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/packages_redhat7.toml"})
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
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/packages_redhat6.toml"})
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
