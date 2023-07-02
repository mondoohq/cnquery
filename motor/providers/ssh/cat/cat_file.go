package cat

import (
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"strings"

	"errors"
	"github.com/kballard/go-shellquote"
)

func NewFile(catfs *Fs, path string, useBase64encoding bool) *File {
	return &File{catfs: catfs, path: path, useBase64encoding: useBase64encoding}
}

type File struct {
	catfs             *Fs
	buf               *bytes.Buffer
	path              string
	useBase64encoding bool
}

func (f *File) readContent() (*bytes.Buffer, error) {
	// we need shellquote to escape filenames with spaces
	catCmd := shellquote.Join("cat", f.path)
	if f.useBase64encoding {
		catCmd = catCmd + " | base64"
	}

	cmd, err := f.catfs.commandRunner.RunCommand(catCmd)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	if f.useBase64encoding {
		data, err = base64.StdEncoding.DecodeString(string(data))
		if err != nil {
			return nil, errors.Join(err, errors.New("could not decode base64 data stream"))
		}
	}

	return bytes.NewBuffer(data), nil
}

func (f *File) Close() error {
	return nil
}

func (f *File) Name() string {
	return f.path
}

func (f *File) Stat() (os.FileInfo, error) {
	return f.catfs.Stat(f.path)
}

func (f *File) Sync() error {
	return nil
}

func (f *File) Truncate(size int64) error {
	return nil
}

func (f *File) Read(b []byte) (n int, err error) {
	if f.buf == nil {
		bufData, err := f.readContent()
		if err != nil {
			return 0, err
		}
		f.buf = bufData
	}
	return f.buf.Read(b)
}

func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *File) Readdir(count int) (res []os.FileInfo, err error) {
	return nil, errors.New("not implemented")
}

func (f *File) Readdirnames(n int) (names []string, err error) {
	// TODO: input n is ignored

	cmd, err := f.catfs.commandRunner.RunCommand("ls -1 '" + f.path + "'")
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	list := strings.Split(strings.TrimSpace(string(data)), "\n")

	if list != nil {
		// filter . and ..
		keep := func(x string) bool {
			if x == "." || x == ".." || x == "" {
				return false
			}
			return true
		}
		m := 0
		for _, x := range list {
			if keep(x) {
				list[m] = x
				m++
			}
		}
		list = list[:m]
	}

	return list, nil
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
