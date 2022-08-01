package scp

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"

	scp_client "github.com/hnakamur/go-scp"
)

// FileOpen copies a file into buffer
// TODO: check handling for directories
// TODO: not suited for large files, we should offload those into a temp directory
func FileOpen(scpClient *scp_client.SCP, path string) (*File, error) {
	// download file
	var buf bytes.Buffer
	_, err := scpClient.Receive(path, &buf)
	if err != nil {
		return nil, err
	}
	return &File{
		buf:       &buf,
		path:      path,
		scpClient: scpClient,
	}, nil
}

type File struct {
	buf       *bytes.Buffer
	path      string
	scpClient *scp_client.SCP
}

func (f *File) Close() error {
	return nil
}

func (f *File) Name() string {
	return f.path
}

func (f *File) Stat() (os.FileInfo, error) {
	return f.scpClient.Receive(f.path, ioutil.Discard)
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
	return f.buf.Write(b)
}

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *File) WriteString(s string) (ret int, err error) {
	return f.buf.WriteString(s)
}
