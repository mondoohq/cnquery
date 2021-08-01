package platformid

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestMacOSMachineId(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/osx_test.toml")
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: filepath})
	require.NoError(t, err)

	lid := MacOSIdProvider{Transport: trans}
	id, err := lid.ID()
	require.NoError(t, err)

	assert.Equal(t, "5c09e2c7-07f2-5bee-be82-7cb70688e55c", id, "machine id is properly detected")
}
