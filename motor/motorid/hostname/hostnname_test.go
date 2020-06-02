package hostname_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestHostnameLinux(t *testing.T) {
	trans, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: "./testdata/hostname_linux.toml"})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	hostame, err := hostname.Hostname(m)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "abefed34cc9c", hostame)
}

func TestHostnameWindows(t *testing.T) {
	trans, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: "./testdata/hostname_windows.toml"})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	hostame, err := hostname.Hostname(m)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "WIN-ABCDEFGVHLD", hostame)
}

func TestHostnameMacos(t *testing.T) {
	trans, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: "./testdata/hostname_macos.toml"})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	hostame, err := hostname.Hostname(m)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "moonshot.local", hostame)
}
