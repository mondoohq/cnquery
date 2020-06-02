package cat_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/motoros/cmd"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/ssh/cat"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestCatFs(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/cat.toml")
	trans, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: filepath})
	require.NoError(t, err)

	cw := &CommandWrapper{
		commandRunner: trans,
		wrapper:       cmd.NewSudo(),
	}

	catfs := cat.New(cw)

	// get file stats
	fi, err := catfs.Stat("/etc/ssh/sshd_config")
	require.NoError(t, err)

	assert.Equal(t, int64(4317), fi.Size())
	assert.Equal(t, false, fi.IsDir())
	assert.Equal(t, os.FileMode(0x180), fi.Mode())
	assert.Equal(t, time.Unix(1590420240, 0), fi.ModTime())

	// fetch file content
	f, err := catfs.Open("/etc/ssh/sshd_config")
	require.NoError(t, err)

	data, err := ioutil.ReadAll(f)
	require.NoError(t, err)

	expected := `X11Forwarding no
PermitRootLogin no
PasswordAuthentication yes
MaxAuthTries 4
UsePAM yes
`

	assert.Equal(t, expected, string(data))
}

type CommandWrapper struct {
	commandRunner cat.CommandRunner
	wrapper       cmd.Wrapper
}

func (cw *CommandWrapper) RunCommand(command string) (*types.Command, error) {
	cmd := cw.wrapper.Build(command)
	return cw.commandRunner.RunCommand(cmd)
}
