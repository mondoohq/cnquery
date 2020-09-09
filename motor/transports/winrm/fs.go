package winrm

import (
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/spf13/afero"
)

func NewWinrmFS() *WinrmFS {
	return &WinrmFS{}
}

// WinrmFS is not implemented yet
type WinrmFS struct{}

func (cat *WinrmFS) Name() string {
	return "Winrm FS"
}

func (cat *WinrmFS) Open(name string) (afero.File, error) {
	return nil, errors.New("not implemented")
}

func (cat *WinrmFS) Stat(name string) (os.FileInfo, error) {
	return nil, errors.New("not implemented")
}

func (cat *WinrmFS) Create(name string) (afero.File, error) {
	return nil, errors.New("not implemented")
}
func (cat *WinrmFS) Mkdir(name string, perm os.FileMode) error {
	return errors.New("not implemented")
}
func (cat *WinrmFS) MkdirAll(path string, perm os.FileMode) error {
	return errors.New("not implemented")
}

func (cat *WinrmFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, errors.New("not implemented")
}

func (cat *WinrmFS) Remove(name string) error {
	return errors.New("not implemented")
}

func (cat *WinrmFS) RemoveAll(path string) error {
	return errors.New("not implemented")
}

func (cat *WinrmFS) Rename(oldname, newname string) error {
	return errors.New("not implemented")
}

func (cat *WinrmFS) Chmod(name string, mode os.FileMode) error {
	return errors.New("not implemented")
}

func (cat *WinrmFS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return errors.New("not implemented")
}
