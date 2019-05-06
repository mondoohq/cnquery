package types

import (
	"os"
	"time"
)

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
