package statutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/types"
	"gotest.tools/assert"
)

func TestLinuxStatCmd(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/linux.toml")
	trans, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: filepath})
	require.NoError(t, err)

	statHelper := New(trans)

	// get file stats
	fi, err := statHelper.Stat("/etc/ssh/sshd_config")
	require.NoError(t, err)

	assert.Equal(t, int64(4317), fi.Size())
	assert.Equal(t, false, fi.IsDir())
	assert.Equal(t, os.FileMode(0x180), fi.Mode())
	assert.Equal(t, time.Unix(1590420240, 0), fi.ModTime())
}
