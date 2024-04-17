// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cat

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
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

func (cat *Fs) Open(name string) (afero.File, error) {
	// NOTE: do not use type here since it does not work well with file names like 'C:\Program Files\New Text Document.txt'
	cmd, err := cat.commandRunner.RunCommand(fmt.Sprintf("powershell -c \"Get-Content '%s'\"", name))
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
	cmd, err := cat.commandRunner.RunCommand(fmt.Sprintf("powershell -c \"Get-Item -LiteralPath '%s' | ConvertTo-JSON\"", name))
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
