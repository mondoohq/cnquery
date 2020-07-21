package machineid_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorapi"
	"go.mondoo.io/mondoo/motor/motorid/machineid"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestGuidWindows(t *testing.T) {
	trans, err := mock.NewFromToml(&motorapi.TransportConfig{Backend: motorapi.TransportBackend_CONNECTION_MOCK, Path: "./testdata/guid_windows.toml"})
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
