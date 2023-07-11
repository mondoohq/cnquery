package cat_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers/os/connection/mock"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/connection/ssh/cat"
)

func TestCatFs(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/cat.toml")
	p, err := mock.New(filepath)
	require.NoError(t, err)

	cw := &CommandWrapper{
		commandRunner: p,
		wrapper:       shared.NewSudo(),
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
	defer f.Close()

	data, err := io.ReadAll(f)
	require.NoError(t, err)

	expected := `X11Forwarding no
PermitRootLogin no
PasswordAuthentication yes
MaxAuthTries 4
UsePAM yes
`
	assert.Equal(t, expected, string(data))

	dir, err := catfs.Open("/etc/ssh")
	require.NoError(t, err)
	defer dir.Close()

	stat, err := dir.Stat()
	require.NoError(t, err)
	assert.Equal(t, true, stat.IsDir())
	files, err := dir.Readdirnames(-1)
	require.NoError(t, err)
	assert.Equal(t, []string{"ssh_config", "ssh_config.d", "ssh_host_ecdsa_key", "ssh_host_ecdsa_key.pub", "ssh_host_ed25519_key", "ssh_host_ed25519_key.pub", "ssh_host_rsa_key", "ssh_host_rsa_key.pub", "sshd_config", "sshd_config.rpmnew"}, files)

	_, err = catfs.Open("/etc/not-there")
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

type CommandWrapper struct {
	commandRunner cat.CommandRunner
	wrapper       shared.Wrapper
}

func (cw *CommandWrapper) RunCommand(command string) (*shared.Command, error) {
	cmd := cw.wrapper.Build(command)
	return cw.commandRunner.RunCommand(cmd)
}
