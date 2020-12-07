package mock

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mondoo.io/mondoo/motor/transports"
)

type FileInfo struct {
	Mode    os.FileMode `toml:"mode"`
	ModTime time.Time   `toml:"time"`
	IsDir   bool        `toml:"isdir"`
	Uid     int64       `toml:"uid"`
	Gid     int64       `toml:"gid"`
}

type MockFileData struct {
	Path     string   `toml:"path"`
	Content  string   `toml:"content"`
	StatData FileInfo `toml:"stat"`
	Enoent   bool     `toml:"enoent"`
}

type MockFile struct {
	data       *MockFileData
	dataReader *strings.Reader
	fs         *mockFS
}

func (mf *MockFile) Name() string {
	return mf.data.Path
}

func (mf *MockFile) Stat() (os.FileInfo, error) {
	if mf.data.Enoent {
		return nil, os.ErrNotExist
	}

	return &transports.FileInfo{
		FName:    filepath.Base(mf.data.Path),
		FSize:    int64(len(mf.data.Content)),
		FModTime: mf.data.StatData.ModTime,
		FMode:    mf.data.StatData.Mode,
		FIsDir:   mf.data.StatData.IsDir,
		Uid:      mf.data.StatData.Uid,
		Gid:      mf.data.StatData.Uid,
	}, nil
}

func (mf *MockFile) reader() *strings.Reader {
	if mf.dataReader == nil {
		mf.dataReader = strings.NewReader(string(mf.data.Content))
	}
	return mf.dataReader
}

func (mf *MockFile) Read(p []byte) (n int, err error) {
	return mf.reader().Read(p)
}

func (mf *MockFile) ReadAt(p []byte, off int64) (n int, err error) {
	return mf.reader().ReadAt(p, off)
}

func (mf *MockFile) Seek(offset int64, whence int) (int64, error) {
	return mf.reader().Seek(offset, whence)
}

func (mf *MockFile) Sync() error {
	return nil
}

func (mf *MockFile) Truncate(size int64) error {
	return errors.New("not implemented")
}

func (mf *MockFile) Write(p []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (mf *MockFile) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (mf *MockFile) WriteString(s string) (ret int, err error) {
	return 0, errors.New("not implemented")
}

func (mf *MockFile) Exists() bool {
	return !mf.data.Enoent
}

func (f *MockFile) Delete() error {
	return errors.New("not implemented")
}

func (f *MockFile) Readdir(n int) ([]os.FileInfo, error) {
	return nil, errors.New("not implemented yet")
}

func (f *MockFile) Readdirnames(n int) ([]string, error) {
	children := []string{}
	path := f.data.Path
	// searches for direct childs of this file
	for k := range f.fs.Files {
		if strings.HasPrefix(k, path) {
			// check if it is only one layer down
			filename := strings.TrimPrefix(k, path)

			// path-seperator is still included, remove it
			filename = strings.TrimPrefix(filename, "/")
			filename = strings.TrimPrefix(filename, "\\")

			if filename == "" || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
				continue
			}
			children = append(children, filename)
		}
		if n > 0 && len(children) > n {
			return children, nil
		}
	}
	return children, nil
}

func (f *MockFile) Close() error {
	// nothing to do
	return nil
}
