package reboot_test

import (
	"path/filepath"
	"testing"

	"go.mondoo.io/mondoo/lumi/resources/reboot"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
	"gotest.tools/assert"
)

func TestRebootOnLinux(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/linux_reboot.toml")
	trans, err := toml.New(&types.Endpoint{Backend: "mock", Path: filepath})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	lb, err := reboot.New(m)
	if err != nil {
		t.Fatal(err)
	}

	required, err := lb.RebootRequired()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, required)
}

func TestRebootOnWindows(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/windows_reboot.toml")
	trans, err := toml.New(&types.Endpoint{Backend: "mock", Path: filepath})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	lb, err := reboot.New(m)
	if err != nil {
		t.Fatal(err)
	}

	required, err := lb.RebootRequired()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, required)
}
