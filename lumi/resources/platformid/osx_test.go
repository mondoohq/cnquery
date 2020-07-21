package platformid

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestMacOSMachineId(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/osx_test.toml")
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: filepath})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	lid := MacOSIdProvider{Motor: m}
	id, err := lid.ID()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "5c09e2c707f25beebe827cb70688e55c", id, "machine id is properly detected")
}
