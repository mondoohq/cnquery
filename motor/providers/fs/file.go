package fs

import (
	"github.com/spf13/afero"
	"os"
)

func NewMountedFile(path string, f afero.File) *FileWrapper {
	return &FileWrapper{
		path:        path,
		mountedFile: f,
	}
}

type FileWrapper struct {
	path        string
	mountedFile afero.File
}

func (f *FileWrapper) Name() string {
	return f.path
}

func (f *FileWrapper) Close() error {
	return f.mountedFile.Close()
}

func (f *FileWrapper) Stat() (os.FileInfo, error) {
	return f.mountedFile.Stat()
}

func (f *FileWrapper) Sync() error {
	return notSupported
}

func (f *FileWrapper) Truncate(size int64) error {
	return notSupported
}

func (f *FileWrapper) Read(b []byte) (n int, err error) {
	return f.mountedFile.Read(b)
}

func (f *FileWrapper) ReadAt(b []byte, off int64) (n int, err error) {
	return f.mountedFile.ReadAt(b, off)
}

func (f *FileWrapper) Readdir(n int) ([]os.FileInfo, error) {
	return f.mountedFile.Readdir(n)
}

func (f *FileWrapper) Readdirnames(n int) ([]string, error) {
	return f.mountedFile.Readdirnames(n)
}

func (f *FileWrapper) Seek(offset int64, whence int) (int64, error) {
	return f.mountedFile.Seek(offset, whence)
}

func (f *FileWrapper) Write(b []byte) (n int, err error) {
	return 0, notSupported
}

func (f *FileWrapper) WriteAt(b []byte, off int64) (n int, err error) {
	return 0, notSupported
}

func (f *FileWrapper) WriteString(s string) (ret int, err error) {
	return 0, notSupported
}
