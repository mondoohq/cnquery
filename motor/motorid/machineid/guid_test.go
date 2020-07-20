package machineid_test

import (
	"testing"

	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorid/machineid"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"gotest.tools/assert"
)

func TestGuidWindows(t *testing.T) {
	trans, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: "./testdata/guid_windows.toml"})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	guid, err := machineid.MachineId(m)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "6BAB78BE-4623-4705-924C-2B22433A4489", guid)
}
