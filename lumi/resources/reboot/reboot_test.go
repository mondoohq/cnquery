package reboot_test

import (
	"path/filepath"
	"testing"

	"go.mondoo.io/mondoo/lumi/resources/reboot"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/types"
	"gotest.tools/assert"
)

func TestRebootOnUbuntu(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/ubuntu_reboot.toml")
	trans, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: filepath})
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

	required, err := lb.RebootPending()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, required)
}

func TestRebootOnRhel(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/redhat_kernel_reboot.toml")
	trans, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: filepath})
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

	required, err := lb.RebootPending()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, required)
}

func TestRebootOnWindows(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/windows_reboot.toml")
	trans, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: filepath})
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

	required, err := lb.RebootPending()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, required)
}
