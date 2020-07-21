package platformid

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorapi"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestLinuxMachineId(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/linux_test.toml")
	trans, err := mock.NewFromToml(&motorapi.TransportConfig{Backend: motorapi.TransportBackend_CONNECTION_MOCK, Path: filepath})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	lid := LinuxIdProvider{Motor: m}
	id, err := lid.ID()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "39827700b8d246eb9446947c573ecff2", id, "machine id is properly detected")
}
