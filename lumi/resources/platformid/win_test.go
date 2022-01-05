package platformid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestGuidWindows(t *testing.T) {
	trans, err := mock.NewFromTomlFile("./testdata/guid_windows.toml")
	require.NoError(t, err)

	lid := WinIdProvider{Transport: trans}
	id, err := lid.ID()
	require.NoError(t, err)

	assert.Equal(t, "6BAB78BE-4623-4705-924C-2B22433A4489", id)
}
