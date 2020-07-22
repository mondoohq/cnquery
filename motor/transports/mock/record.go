package mock

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
)

func NewRecordTransport(trans transports.Transport) (*RecordTransport, error) {
	mock, err := New()
	if err != nil {
		return nil, err
	}

	recordWrapper := &RecordTransport{
		mock:    mock,
		observe: trans,
	}

	return recordWrapper, nil
}

type RecordTransport struct {
	observe transports.Transport
	mock    *Transport
}

func (t *RecordTransport) Watched() transports.Transport {
	return t.observe
}

func (t *RecordTransport) Export() (*TomlData, error) {
	return Export(t.mock)
}

func (t *RecordTransport) ExportData() ([]byte, error) {
	return ExportData(t.mock)
}

func (t *RecordTransport) RunCommand(command string) (*transports.Command, error) {
	cmd, err := t.observe.RunCommand(command)
	if err != nil {
		// we do not record errors yet
		return nil, err
	}

	if cmd != nil {
		stdout := ""
		stderr := ""

		stdoutData, err := ioutil.ReadAll(cmd.Stdout)
		if err == nil {
			stdout = string(stdoutData)
		}
		stderrData, err := ioutil.ReadAll(cmd.Stderr)
		if err == nil {
			stderr = string(stderrData)
		}

		// store command
		t.mock.Commands[command] = &Command{
			Command:    command,
			Stdout:     stdout,
			Stderr:     stderr,
			ExitStatus: cmd.ExitStatus,
		}
	}

	// read command from mock
	return t.mock.RunCommand(command)
}

func (t *RecordTransport) FS() afero.Fs {
	fs := t.observe.FS()
	return NewRecordFS(fs, t.mock.Fs)
}

func (t *RecordTransport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return t.observe.FileInfo(path)
}

func (t *RecordTransport) Capabilities() transports.Capabilities {
	return t.observe.Capabilities()
}

func (t *RecordTransport) Close() {
	t.observe.Close()
}

func NewRecordFS(observe afero.Fs, mockfs *mockFS) *recordFS {
	return &recordFS{
		observe: observe,
		mock:    mockfs,
	}
}

type recordFS struct {
	observe afero.Fs
	mock    *mockFS
}

func (fs recordFS) Name() string {
	return fs.observe.Name() + " (recording)"
}

func (fs recordFS) Create(name string) (afero.File, error) {
	return fs.observe.Create(name)
}

func (fs recordFS) Mkdir(name string, perm os.FileMode) error {
	return fs.observe.Mkdir(name, perm)
}

func (fs recordFS) MkdirAll(path string, perm os.FileMode) error {
	return fs.observe.MkdirAll(path, perm)
}

func (fs recordFS) Open(name string) (afero.File, error) {
	enonet := false
	content := ""

	f, err := fs.observe.Open(name)
	if err == os.ErrNotExist {
		enonet = true
	} else if err != nil {
		return nil, err
	} else {
		data, err := ioutil.ReadAll(f)
		defer f.Close()
		if err != nil {
			return nil, err
		}
		content = string(data)
	}

	fMock, ok := fs.mock.Files[name]
	if !ok {
		fMock = &MockFileData{}
	}

	fMock.Content = content
	fMock.Path = name
	fMock.Enoent = enonet

	fs.mock.Files[name] = fMock

	// return data from mockfs
	return fs.mock.Open(name)
}

func (fs recordFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return fs.observe.OpenFile(name, flag, perm)
}

func (fs recordFS) Remove(name string) error {
	return fs.observe.Remove(name)
}

func (fs recordFS) RemoveAll(path string) error {
	return fs.observe.RemoveAll(path)
}

func (fs recordFS) Rename(oldname, newname string) error {
	return fs.observe.Rename(oldname, newname)
}

func (fs recordFS) Stat(name string) (os.FileInfo, error) {
	return fs.observe.Stat(name)
}

// func (fs recordFS) Lstat(p string) (os.FileInfo, error) {
// 	return fs.observe.Lstat(p)
// }

func (fs recordFS) Chmod(name string, mode os.FileMode) error {
	return fs.observe.Chmod(name, mode)
}

func (fs recordFS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return fs.observe.Chtimes(name, atime, mtime)
}

// func (fs recordFS) Glob(pattern string) ([]string, error) {
// 	return fs.observe.Glob(pattern)
// }
