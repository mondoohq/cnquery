package fsutil

import (
	"errors"
	"os"
	"time"

	"github.com/spf13/afero"
)

type NoFs struct{}

var notImplementedError = errors.New("not implemented")

func (NoFs) Create(name string) (afero.File, error) {
	return nil, notImplementedError
}

func (NoFs) Mkdir(name string, perm os.FileMode) error {
	return notImplementedError
}

func (NoFs) MkdirAll(path string, perm os.FileMode) error {
	return notImplementedError
}

func (NoFs) Open(name string) (afero.File, error) {
	return nil, notImplementedError
}

func (NoFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, notImplementedError
}

func (NoFs) Remove(name string) error {
	return notImplementedError
}

func (NoFs) RemoveAll(path string) error {
	return notImplementedError
}

func (NoFs) Rename(oldname, newname string) error {
	return notImplementedError
}

func (NoFs) Stat(name string) (os.FileInfo, error) {
	return nil, notImplementedError
}

func (NoFs) Name() string {
	return "nofs"
}

func (NoFs) Chmod(name string, mode os.FileMode) error {
	return notImplementedError
}

func (NoFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return notImplementedError
}
