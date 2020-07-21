package reboot_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/reboot"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestRebootOnUbuntu(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/ubuntu_reboot.toml")
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: filepath})
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
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: filepath})
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
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: filepath})
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
