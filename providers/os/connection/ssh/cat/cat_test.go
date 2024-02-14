// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

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
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/connection/ssh/cat"
)

func TestCatFs(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/cat.toml")
	p, err := mock.New(0, filepath, &inventory.Asset{})
	require.NoError(t, err)

	flags := map[string]*llx.Primitive{
		"sudo": llx.BoolPrimitive(true),
	}

	cw := &CommandWrapper{
		commandRunner: p,
		sudo:          shared.ParseSudo(flags),
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
	sudo          *inventory.Sudo
}

func (cw *CommandWrapper) RunCommand(command string) (*shared.Command, error) {
	cmd := shared.BuildSudoCommand(cw.sudo, command)
	return cw.commandRunner.RunCommand(cmd)
}
