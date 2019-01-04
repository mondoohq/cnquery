package mock

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"go.mondoo.io/mondoo/motor/motorutil"
	"go.mondoo.io/mondoo/motor/types"
)

type MockFile struct {
	file *File
}

func (mf *MockFile) Name() string {
	return mf.file.Path
}

func (mf *MockFile) Stat() (os.FileInfo, error) {
	if mf.file.Enoent {
		return nil, errors.New("no such file or directory")
	}

	f := mf.file
	stat := types.FileInfo{FSize: int64(len(f.Content)), FModTime: f.Stat.ModTime, FMode: f.Stat.Mode, FIsDir: f.Stat.IsDir}
	return &stat, nil
}

// TODO, support directory streaming
func (mf *MockFile) Tar() (io.ReadCloser, error) {
	if mf.file.Enoent {
		return nil, errors.New("no such file or directory")
	}

	f := mf.file
	fReader := ioutil.NopCloser(strings.NewReader(string(f.Content)))

	stat, err := mf.Stat()
	if err != nil {
		return nil, errors.New("could not retrieve file stats")
	}

	// create a pipe
	tarReader, tarWriter := io.Pipe()

	// convert raw stream to tar stream
	go motorutil.StreamFileAsTar(mf.Name(), stat, fReader, tarWriter)

	// return the reader
	return tarReader, nil
}

func (mf *MockFile) Open() (types.FileStream, error) {
	if mf.file.Enoent {
		return nil, errors.New("no such file or directory")
	}

	f := mf.file
	fReader := ioutil.NopCloser(strings.NewReader(string(f.Content)))
	return fReader, nil
}

func (mf *MockFile) HashMd5() (string, error) {
	f := mf.file
	h := md5.New()
	h.Write([]byte(f.Content))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (mf *MockFile) HashSha256() (string, error) {
	f := mf.file
	h := sha256.New()
	h.Write([]byte(f.Content))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (mf *MockFile) Exists() bool {
	return !mf.file.Enoent
}

func (f *MockFile) Delete() error {
	return errors.New("not implemented")
}

func (f *MockFile) Readdir(n int) ([]os.FileInfo, error) {
	return nil, errors.New("not implemented yet")
}

func (f *MockFile) Readdirnames(n int) ([]string, error) {
	return nil, errors.New("not implemented yet")
}
