package mock_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestGlobCommand(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/mock.toml")
	trans, err := toml.New(&types.Endpoint{Backend: "mock", Path: filepath})
	assert.Equal(t, nil, err, "should create mock without error")

	filesystem := trans.Fs
	matches, err := filesystem.Glob("*ssh/*_config")

	assert.True(t, len(matches) == 1)
	assert.Contains(t, matches, "/etc/ssh/sshd_config")
}

func TestLoadFile(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/mock.toml")
	trans, err := toml.New(&types.Endpoint{Backend: "mock", Path: filepath})
	assert.Equal(t, nil, err, "should create mock without error")

	f, err := trans.File("/etc/os-release")
	assert.Nil(t, err)

	data, err := ioutil.ReadAll(f)
	assert.Nil(t, err)

	assert.Equal(t, 382, len(data))
}
