package reboot_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/reboot"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorapi"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestRebootLinux(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/ubuntu_reboot.toml")
	trans, err := mock.NewFromToml(&motorapi.TransportConfig{Backend: motorapi.TransportBackend_CONNECTION_MOCK, Path: filepath})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	lb := reboot.DebianReboot{Motor: m}
	required, err := lb.RebootPending()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, required)
}

func TestNoRebootLinux(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/ubuntu_noreboot.toml")
	trans, err := mock.NewFromToml(&motorapi.TransportConfig{Backend: motorapi.TransportBackend_CONNECTION_MOCK, Path: filepath})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	lb := reboot.DebianReboot{Motor: m}
	required, err := lb.RebootPending()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, false, required)
}
