// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cat

import (
	"bytes"
	"io"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/resources/powershell"
)

type CommandRunner interface {
	RunCommand(command string) (*shared.Command, error)
}

func New(cmdRunner CommandRunner) *Fs {
	return &Fs{
		commandRunner: cmdRunner,
	}
}

type Fs struct {
	commandRunner CommandRunner
}

func (cat *Fs) Name() string {
	return "Winrm Cat FS"
}

// The two scripts below are handed to powershell as an encoded command
// rather than as a quoted command line. A plain `powershell -c "... '<name>'"`
// nests two parsers, the command line and then powershell itself, so a name
// has to survive both. Encoding removes the outer layer entirely and
// SingleQuote covers the inner one.

func getContentScript(name string) string {
	return powershell.Encode("Get-Content -LiteralPath " + powershell.SingleQuote(name))
}

func getItemScript(name string) string {
	return powershell.Encode("Get-Item -LiteralPath " + powershell.SingleQuote(name) + " | ConvertTo-JSON")
}

func (cat *Fs) Open(name string) (afero.File, error) {
	// NOTE: do not use type here since it does not work well with file names like 'C:\Program Files\New Text Document.txt'
	cmd, err := cat.commandRunner.RunCommand(getContentScript(name))
	if err != nil {
		return nil, err
	}

	if cmd.ExitStatus != 0 {
		return nil, os.ErrNotExist
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	return NewFile(name, bytes.NewBuffer(data)), nil
}

func (cat *Fs) Stat(name string) (os.FileInfo, error) {
	cmd, err := cat.commandRunner.RunCommand(getItemScript(name))
	if err != nil {
		return nil, err
	}

	if cmd.ExitStatus != 0 {
		return nil, os.ErrNotExist
	}

	item, err := ParseGetItem(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	return &fileStat{
		name:           item.BaseName,
		FileSize:       item.Length,
		FileAttributes: item.Attributes,
		CreationTime:   powershell.PSJsonTimestamp(item.CreationTime),
		LastAccessTime: powershell.PSJsonTimestamp(item.LastAccessTime),
		LastWriteTime:  powershell.PSJsonTimestamp(item.LastWriteTime),
	}, nil
}

var NotImplemented = errors.New("not implemented")

func (cat *Fs) Create(name string) (afero.File, error) {
	return nil, errors.New("not implemented")
}

func (cat *Fs) Mkdir(name string, perm os.FileMode) error {
	return NotImplemented
}

func (cat *Fs) MkdirAll(path string, perm os.FileMode) error {
	return NotImplemented
}

func (cat *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, NotImplemented
}

func (cat *Fs) Remove(name string) error {
	return NotImplemented
}

func (cat *Fs) RemoveAll(path string) error {
	return NotImplemented
}

func (cat *Fs) Rename(oldname, newname string) error {
	return NotImplemented
}

func (cat *Fs) Chmod(name string, mode os.FileMode) error {
	return NotImplemented
}

func (cat *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return NotImplemented
}

func (cat *Fs) Chown(name string, uid, gid int) error {
	return NotImplemented
}
