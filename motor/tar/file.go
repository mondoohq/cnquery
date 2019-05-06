package tar

import (
	"archive/tar"
	"bufio"
	"errors"
	"os"
)

type File struct {
	path   string
	header *tar.Header
	Fs     *FS
	reader *bufio.Reader
}

func (f *File) Name() string {
	return f.path
}

func (f *File) Close() error {
	return nil
}

func (f *File) Stat() (os.FileInfo, error) {
	return f.Fs.stat(f.header)
}

func (f *File) Sync() error {
	return errors.New("not implemented")
}

func (f *File) Truncate(size int64) error {
	return errors.New("not implemented")
}

func (f *File) Read(b []byte) (n int, err error) {
	if f.reader == nil {
		return 0, errors.New("no tar data available")
	}
	return f.reader.Read(b)
}

func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	return 0, errors.New("not implemented yet")
}

func (f *File) Readdir(n int) ([]os.FileInfo, error) {
	return nil, errors.New("not implemented yet")
}

func (f *File) Readdirnames(n int) ([]string, error) {
	return nil, errors.New("not implemented yet")
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
