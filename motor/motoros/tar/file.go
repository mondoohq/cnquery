package tar

import (
	"archive/tar"
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	fi := []os.FileInfo{}
	// search all child items
	for k := range f.Fs.FileMap {
		entry := f.Fs.FileMap[k].Name
		// log.Debug().Str("entry", Abs(entry)).Str("path", f.path).Msg("iteratte path")
		if strings.HasPrefix(Abs(entry), f.path) {
			fi = append(fi)
		}
	}
	return fi, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	fi := []string{}
	// search all child items
	for k := range f.Fs.FileMap {
		entry := f.Fs.FileMap[k].Name

		if strings.HasPrefix(Abs(entry), f.path) {
			// extract file name
			rel, err := filepath.Rel(f.path, Abs(entry))
			if err != nil {
				return nil, err
			}

			// skip own path
			if rel == "." {
				continue
			}

			// log.Debug().Str("entry", Abs(entry)).Str("path", f.path).Str("rel", rel).Msg("rel")
			fi = append(fi, rel)
		}
	}
	return fi, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("seek not implemented")
}

func (f *File) Write(b []byte) (n int, err error) {
	return 0, errors.New("write not implemented")
}

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	return 0, errors.New("writeat not implemented")
}

func (f *File) WriteString(s string) (ret int, err error) {
	return 0, errors.New("writestring not implemented")
}
