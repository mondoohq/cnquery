package mock

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	os_provider "go.mondoo.com/cnquery/motor/providers/os"
)

type FileInfo struct {
	Mode    os.FileMode `toml:"mode"`
	ModTime time.Time   `toml:"time"`
	IsDir   bool        `toml:"isdir"`
	Uid     int64       `toml:"uid"`
	Gid     int64       `toml:"gid"`
	Size    int64       `toml:"size"`
}

type MockFileData struct {
	Path string `toml:"path"`

	StatData FileInfo `toml:"stat"`
	Enoent   bool     `toml:"enoent"`
	// Holds the file content
	Data []byte `toml:"data"`
	// Plain String response (simpler user usage, will not be used for automated recording)
	Content string `toml:"content"`
}

type ReadAtSeeker interface {
	io.Reader
	io.Seeker
	io.ReaderAt
}

type MockFile struct {
	data       *MockFileData
	dataReader ReadAtSeeker
	fs         *mockFS
}

func (mf *MockFile) Name() string {
	return mf.data.Path
}

func (mf *MockFile) Stat() (os.FileInfo, error) {
	if mf.data.Enoent {
		return nil, os.ErrNotExist
	}

	// fallback in case the size information is missing, eg. older mock files
	var size int64
	if mf.data.StatData.Size > 0 {
		size = mf.data.StatData.Size
	} else if mf.data.StatData.Size == 0 && len(mf.data.Data) > 0 {
		size = int64(len(mf.data.Data))
	} else if mf.data.StatData.Size == 0 && len(mf.data.Content) > 0 {
		size = int64(len(mf.data.Content))
	}

	return &os_provider.FileInfo{
		FName:    filepath.Base(mf.data.Path),
		FSize:    size,
		FModTime: mf.data.StatData.ModTime,
		FMode:    mf.data.StatData.Mode,
		FIsDir:   mf.data.StatData.IsDir,
		Uid:      mf.data.StatData.Uid,
		Gid:      mf.data.StatData.Uid,
	}, nil
}

func (mf *MockFile) reader() ReadAtSeeker {
	// if binary data was provided, we ignore the string data
	if mf.dataReader == nil && len(mf.data.Data) > 0 {
		mf.dataReader = bytes.NewReader(mf.data.Data)
	} else if mf.dataReader == nil {
		mf.dataReader = strings.NewReader(mf.data.Content)
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
	children := []os.FileInfo{}
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

			// fetch file stats
			fsInfo, err := f.fs.Stat(k)
			if err != nil {
				return nil, errors.New("cannot find file in mock index: " + k)
			}

			children = append(children, fsInfo)
		}
		if n > 0 && len(children) > n {
			return children, nil
		}
	}
	return children, nil
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
