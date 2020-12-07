package mock

import (
	"errors"
	"os"
	"time"

	"github.com/gobwas/glob"
	"github.com/spf13/afero"
)

type mockFS struct {
	Files map[string]*MockFileData
}

func NewMockFS() *mockFS {
	return &mockFS{
		Files: make(map[string]*MockFileData),
	}
}

func (f mockFS) Name() string {
	return "mockfs"
}

func (fs mockFS) Create(name string) (afero.File, error) {
	return nil, errors.New("not implemented")
}

func (fs mockFS) Mkdir(name string, perm os.FileMode) error {
	return errors.New("not implemented")
}

func (fs mockFS) MkdirAll(path string, perm os.FileMode) error {
	return errors.New("not implemented")
}

func (fs mockFS) Open(name string) (afero.File, error) {
	data, ok := fs.Files[name]
	if !ok || data.Enoent {
		return nil, os.ErrNotExist
	}

	return &MockFile{
		data: data,
		fs:   &fs,
	}, nil
}

func (fs mockFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, errors.New("not implemented")
}

func (fs mockFS) Remove(name string) error {
	delete(fs.Files, name)
	return nil
}

func (fs mockFS) RemoveAll(path string) error {
	return errors.New("not implemented")
}

func (fs mockFS) Rename(oldname, newname string) error {
	if oldname == newname {
		return nil
	}

	f, ok := fs.Files[oldname]
	if !ok {
		return os.ErrNotExist
	}

	fs.Files[newname] = f
	return nil
}

func (fs mockFS) Stat(name string) (os.FileInfo, error) {
	data, ok := fs.Files[name]
	if !ok {
		return nil, os.ErrNotExist
	}

	f := &MockFile{
		data: data,
		fs:   &fs,
	}

	return f.Stat()
}

func (fs mockFS) Lstat(p string) (os.FileInfo, error) {
	return nil, errors.New("not implemented")
}

func (fs mockFS) Chmod(name string, mode os.FileMode) error {
	return errors.New("not implemented")
}

func (fs mockFS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return errors.New("not implemented")
}

func (fs mockFS) Glob(pattern string) ([]string, error) {
	matches := []string{}

	g, err := glob.Compile(pattern)
	if err != nil {
		return matches, err
	}

	for k := range fs.Files {
		if g.Match(k) {
			matches = append(matches, k)
		}
	}

	return matches, nil
}
