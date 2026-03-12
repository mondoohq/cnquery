// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	sshcat "go.mondoo.com/mql/v13/providers/os/connection/ssh/cat"
	"go.mondoo.com/mql/v13/utils/syncx"
)

type fileCommandWrapper struct {
	commandRunner sshcat.CommandRunner
	sudo          *inventory.Sudo
	commands      []string
}

func (cw *fileCommandWrapper) RunCommand(command string) (*shared.Command, error) {
	cmd := shared.BuildSudoCommand(cw.sudo, command)
	cw.commands = append(cw.commands, cmd)
	return cw.commandRunner.RunCommand(cmd)
}

type sudoCatConnection struct {
	asset  *inventory.Asset
	runner *fileCommandWrapper
	fs     afero.Fs
}

func newSudoCatConnection(t *testing.T) *sudoCatConnection {
	t.Helper()

	fixturePath, err := filepath.Abs("../connection/ssh/cat/testdata/cat.toml")
	require.NoError(t, err)

	asset := &inventory.Asset{
		Platform: &inventory.Platform{
			Name:    "ubuntu",
			Version: "22.04",
			Family:  []string{"ubuntu", "linux"},
		},
	}
	mockConn, err := mock.New(0, asset, mock.WithPath(fixturePath))
	require.NoError(t, err)

	flags := map[string]*llx.Primitive{
		"sudo": llx.BoolPrimitive(true),
	}
	runner := &fileCommandWrapper{
		commandRunner: mockConn,
		sudo:          shared.ParseSudo(flags),
	}

	return &sudoCatConnection{
		asset:  asset,
		runner: runner,
		fs:     sshcat.New(runner),
	}
}

func (c *sudoCatConnection) ID() uint32                         { return 0 }
func (c *sudoCatConnection) ParentID() uint32                   { return 0 }
func (c *sudoCatConnection) Name() string                       { return "sudo-cat-test" }
func (c *sudoCatConnection) Type() shared.ConnectionType        { return shared.Type_SSH }
func (c *sudoCatConnection) Asset() *inventory.Asset            { return c.asset }
func (c *sudoCatConnection) UpdateAsset(asset *inventory.Asset) { c.asset = asset }
func (c *sudoCatConnection) Capabilities() shared.Capabilities {
	return shared.Capability_File | shared.Capability_RunCommand
}
func (c *sudoCatConnection) RunCommand(command string) (*shared.Command, error) {
	return c.runner.RunCommand(command)
}
func (c *sudoCatConnection) FileSystem() afero.Fs { return c.fs }
func (c *sudoCatConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	stat, err := (&afero.Afero{Fs: c.fs}).Stat(path)
	if err != nil {
		return shared.FileInfoDetails{}, err
	}
	sysStat, ok := stat.Sys().(*shared.FileInfo)
	if !ok {
		return shared.FileInfoDetails{}, errors.New("unexpected stat type")
	}

	return shared.FileInfoDetails{
		Mode: shared.FileModeDetails{FileMode: stat.Mode()},
		Size: stat.Size(),
		Uid:  sysStat.Uid,
		Gid:  sysStat.Gid,
	}, nil
}

func TestFileExistsSharesStatMetadataLoad(t *testing.T) {
	conn := newSudoCatConnection(t)
	runtime := &plugin.Runtime{
		Connection: conn,
		Resources:  &syncx.Map[plugin.Resource]{},
	}

	raw, err := CreateResource(runtime, "file", map[string]*llx.RawData{
		"path": llx.StringData("/etc/ssh/sshd_config"),
	})
	require.NoError(t, err)

	file := raw.(*mqlFile)
	exists := file.GetExists()
	require.NoError(t, exists.Error)
	require.True(t, exists.Data)

	permissions := file.GetPermissions()
	require.NoError(t, permissions.Error)

	size := file.GetSize()
	require.NoError(t, size.Error)

	assert.Equal(t, 1, countRecordedCommands(conn.runner.commands, "sudo uname -s"))
	assert.Equal(t, 1, countRecordedCommands(conn.runner.commands, "sudo test -e /etc/ssh/sshd_config"))
	assert.Equal(t, 1, countRecordedCommands(conn.runner.commands, "sudo stat -L /etc/ssh/sshd_config -c '%s.%f.%u.%g.%X.%Y.%C'"))
}

func countRecordedCommands(commands []string, target string) int {
	count := 0
	for _, command := range commands {
		if command == target {
			count++
		}
	}
	return count
}
