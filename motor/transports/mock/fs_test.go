package mock_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestGlobCommand(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/mock.toml")
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: filepath})
	assert.Equal(t, nil, err, "should create mock without error")

	filesystem := trans.Fs
	matches, err := filesystem.Glob("*ssh/*_config")

	assert.True(t, len(matches) == 1)
	assert.Contains(t, matches, "/etc/ssh/sshd_config")
}

func TestLoadFile(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/mock.toml")
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: filepath})
	assert.Equal(t, nil, err, "should create mock without error")

	f, err := trans.FS().Open("/etc/os-release")
	assert.Nil(t, err)

	data, err := ioutil.ReadAll(f)
	assert.Nil(t, err)

	assert.Equal(t, 382, len(data))
}

func TestReadDirnames(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/mock.toml")
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: filepath})
	require.NoError(t, err)

	dir, err := trans.FS().Open("/sys/class/dmi/id")
	require.NoError(t, err)
	stat, err := dir.Stat()
	require.NoError(t, err)
	assert.True(t, stat.IsDir())

	names, err := dir.Readdirnames(100)
	require.NoError(t, err)

	assert.Equal(t, 2, len(names))
	assert.Contains(t, names, "bios_vendor")
	assert.Contains(t, names, "bios_date")
}
