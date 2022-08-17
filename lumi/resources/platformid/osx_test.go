package platformid

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestMacOSMachineId(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/osx_test.toml")
	provider, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	lid := MacOSIdProvider{provider: provider}
	id, err := lid.ID()
	require.NoError(t, err)

	assert.Equal(t, "5c09e2c7-07f2-5bee-be82-7cb70688e55c", id, "machine id is properly detected")
}
