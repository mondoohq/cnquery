package mock_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/motoros/fsutil"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestMockCommand(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/mock.toml")
	trans, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: filepath})
	assert.Equal(t, nil, err, "should create mock without error")

	cmd, err := trans.RunCommand("ls /")

	if assert.NotNil(t, cmd) {
		assert.Equal(t, nil, err, "should execute without error")
		stdoutContent, _ := ioutil.ReadAll(cmd.Stdout)
		assert.Equal(t, "bin  boot  dev  etc  home  lib  lib64  media  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var", string(stdoutContent), "output should be correct")
		stderrContent, _ := ioutil.ReadAll(cmd.Stdout)
		assert.Equal(t, "", string(stderrContent), "output should be correct")
	}
}

func TestMockFile(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/mock.toml")
	trans, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: filepath})
	assert.Equal(t, nil, err, "should create mock without error")

	f, err := trans.File("/etc/ssh/sshd_config")
	assert.Nil(t, err, "should execute without error")
	assert.NotNil(t, f)
	defer f.Close()

	afutil := afero.Afero{Fs: trans.FS()}
	afutil.Exists(f.Name())

	p := f.Name()
	assert.Equal(t, "/etc/ssh/sshd_config", p, "path should be correct")

	stat, err := f.Stat()
	assert.Equal(t, int64(3218), stat.Size(), "should read file size")
	assert.Nil(t, err, "should execute without error")

	content, err := afutil.ReadFile(f.Name())
	assert.Equal(t, nil, err, "should execute without error")
	assert.Equal(t, 3218, len(content), "should read the full content")

	// reset reader
	f.Seek(0, 0)
	sha, err := fsutil.Sha256(f)
	assert.Equal(t, "be0e5cb10ab5b8bdce48198199c5facad387ffa7a7b0098b6b31909b3fafc413", sha, "sha256 output should be correct")
	assert.Nil(t, err, "should execute without error")

	// reset reader
	f.Seek(0, 0)
	md5, err := fsutil.Md5(f)
	assert.Equal(t, "c18b98e3ae04f26e62ed52a3d76db5e9", md5, "md5 output should be correct")
	assert.Nil(t, err, "should execute without error")

}
