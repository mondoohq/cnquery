package reboot_test

import (
	"path/filepath"
	"testing"

	"go.mondoo.io/mondoo/lumi/resources/reboot"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"gotest.tools/assert"
)

func TestRhelKernelLatest(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/redhat_kernel_reboot.toml")
	trans, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: filepath})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	lb := reboot.RpmNewestKernel{Motor: m}
	required, err := lb.RebootPending()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, required)
}

func TestAmznContainerWithoutKernel(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/amzn_kernel_container.toml")
	trans, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: filepath})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	lb := reboot.RpmNewestKernel{Motor: m}
	required, err := lb.RebootPending()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, false, required)
}

func TestAmznEc2Kernel(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/amzn_kernel_ec2.toml")
	trans, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: filepath})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	lb := reboot.RpmNewestKernel{Motor: m}
	required, err := lb.RebootPending()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, false, required)
}
