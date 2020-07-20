package statutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"gotest.tools/assert"
)

func TestLinuxStatCmd(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/linux.toml")
	trans, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: filepath})
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

func TestOpenbsdStatCmd(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/openbsd.toml")
	trans, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: filepath})
	require.NoError(t, err)

	statHelper := New(trans)

	// get file stats
	fi, err := statHelper.Stat("/etc/ssh/sshd_config")
	require.NoError(t, err)

	assert.Equal(t, int64(2259), fi.Size())
	assert.Equal(t, false, fi.IsDir())
	assert.Equal(t, "-rw-r--r--", fi.Mode().String())
	assert.Equal(t, time.Unix(1592996018, 0), fi.ModTime())
	assert.Equal(t, int64(0), fi.Sys().(*transports.FileInfo).Uid)
	assert.Equal(t, int64(0), fi.Sys().(*transports.FileInfo).Gid)
}
