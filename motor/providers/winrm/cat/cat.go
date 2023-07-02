package cat

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"errors"
	"github.com/spf13/afero"
	os_provider "go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
)

type CommandRunner interface {
	RunCommand(command string) (*os_provider.Command, error)
}

func New(cmdRunner CommandRunner) *CatFs {
	return &CatFs{
		commandRunner: cmdRunner,
	}
}

type CatFs struct {
	commandRunner CommandRunner
}

func (cat *CatFs) Name() string {
	return "Winrm Cat FS"
}

func (cat *CatFs) Open(name string) (afero.File, error) {
	// NOTE: do not use type here since it does not work well with file names like 'C:\Program Files\New Text Document.txt'
	cmd, err := cat.commandRunner.RunCommand(fmt.Sprintf("powershell -c \"Get-Content '%s'\"", name))
	if err != nil {
		return nil, err
	}

	if cmd.ExitStatus != 0 {
		return nil, os.ErrNotExist
	}

	data, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	return NewFile(name, bytes.NewBuffer(data)), nil
}

func (cat *CatFs) Stat(name string) (os.FileInfo, error) {
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

func (cat *CatFs) Create(name string) (afero.File, error) {
	return nil, errors.New("not implemented")
}

func (cat *CatFs) Mkdir(name string, perm os.FileMode) error {
	return errors.New("not implemented")
}

func (cat *CatFs) MkdirAll(path string, perm os.FileMode) error {
	return errors.New("not implemented")
}

func (cat *CatFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, errors.New("not implemented")
}

func (cat *CatFs) Remove(name string) error {
	return errors.New("not implemented")
}

func (cat *CatFs) RemoveAll(path string) error {
	return errors.New("not implemented")
}

func (cat *CatFs) Rename(oldname, newname string) error {
	return errors.New("not implemented")
}

func (cat *CatFs) Chmod(name string, mode os.FileMode) error {
	return errors.New("not implemented")
}

func (cat *CatFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return errors.New("not implemented")
}

func (cat *CatFs) Chown(name string, uid, gid int) error {
	return errors.New("not implemented")
}
