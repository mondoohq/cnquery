package cat

import (
	"bytes"
	"errors"
	"os"
)

func NewFile(name string, buf *bytes.Buffer) *File {
	return &File{path: name, buf: buf}
}

type File struct {
	buf  *bytes.Buffer
	path string
}

func (f *File) Close() error {
	return nil
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
	return f.buf.Read(b)
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
