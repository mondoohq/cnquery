package platformid

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestLinuxMachineId(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/linux_test.toml")
	trans, err := mock.NewFromTomlFile(filepath)
	require.NoError(t, err)

	lid := LinuxIdProvider{Transport: trans}
	id, err := lid.ID()
	require.NoError(t, err)

	assert.Equal(t, "39827700b8d246eb9446947c573ecff2", id, "machine id is properly detected")
}
