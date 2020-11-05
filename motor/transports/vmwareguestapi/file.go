package vmwareguestapi

import (
	"errors"
	"io"
	"os"
)

func NewFile(name string, rc io.ReadCloser) *File {
	return &File{path: name, rc: rc}
}

type File struct {
	rc   io.ReadCloser
	path string
}

func (f *File) Close() error {
	return f.rc.Close()
}

func (f *File) Name() string {
	return f.path
}

func (f *File) Stat() (os.FileInfo, error) {
	return nil, errors.New("not implemented")
}

func (f *File) Sync() error {
	return nil
}

func (f *File) Truncate(size int64) error {
	return nil
}

func (f *File) Read(b []byte) (n int, err error) {
	return f.rc.Read(b)
}

func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *File) Readdir(count int) (res []os.FileInfo, err error) {
	return nil, errors.New("not implemented")
}

func (f *File) Readdirnames(n int) (names []string, err error) {
	return nil, errors.New("not implemented")
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("not implemented")
}

func (f *File) Write(b []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *File) WriteString(s string) (ret int, err error) {
	return 0, errors.New("not implemented")
}
