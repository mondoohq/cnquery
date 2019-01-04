package toml

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/mock"
	"go.mondoo.io/mondoo/motor/motorutil"
	"go.mondoo.io/mondoo/motor/types"
)

func TestMockCommand(t *testing.T) {

	filepath, _ := filepath.Abs("./mock.toml")
	trans, err := New(&types.Endpoint{Backend: "mock", Path: filepath})
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

	filepath, _ := filepath.Abs("./mock.toml")
	trans, err := New(&types.Endpoint{Backend: "mock", Path: filepath})
	assert.Equal(t, nil, err, "should create mock without error")

	f, err := trans.File("/etc/ssh/sshd_config")

	if assert.NotNil(t, f) {
		assert.Equal(t, nil, err, "should execute without error")

		p := f.Name()
		assert.Equal(t, "/etc/ssh/sshd_config", p, "path should be correct")

		stat, err := f.Stat()
		assert.Equal(t, int64(3218), stat.Size(), "should read file size")
		assert.Equal(t, nil, err, "should execute without error")

		sha, err := f.(*mock.MockFile).HashSha256()
		assert.Equal(t, "be0e5cb10ab5b8bdce48198199c5facad387ffa7a7b0098b6b31909b3fafc413", sha, "sha256 output should be correct")
		assert.Equal(t, nil, err, "should execute without error")

		md5, err := f.(*mock.MockFile).HashMd5()
		assert.Equal(t, "c18b98e3ae04f26e62ed52a3d76db5e9", md5, "md5 output should be correct")
		assert.Equal(t, nil, err, "should execute without error")

		content, err := motorutil.ReadFile(f)
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, 3218, len(content), "should read the full content")
	}
}
