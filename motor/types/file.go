package types

import (
	"io"
	"os"
	"time"
)

type FileStream interface {
	Read(b []byte) (n int, err error)
	Close() error
}

type File interface {
	Name() string

	Exists() bool

	// returns the raw stream of a file content, directories will not be supported
	Open() (FileStream, error)

	Stat() (os.FileInfo, error)
	Readdir(n int) ([]os.FileInfo, error)
	Readdirnames(n int) (names []string, err error)

	// send streams as tar, it is more efficient and allows directory streaming
	// We are using io.ReadCloser instead of golang File interface, since
	// we may not have random access to the file system
	Tar() (io.ReadCloser, error)
}

type FileInfo struct {
	FName    string
	FSize    int64
	FIsDir   bool
	FModTime time.Time
	FMode    os.FileMode
}

func (f *FileInfo) Name() string {
	return f.FName
}

func (f *FileInfo) Size() int64 {
	return f.FSize
}

func (f *FileInfo) Mode() os.FileMode {
	return f.FMode
}

func (f *FileInfo) ModTime() time.Time {
	return f.FModTime
}

func (f *FileInfo) IsDir() bool {
	return f.FIsDir
}

func (f *FileInfo) Sys() interface{} {
	return nil
}
