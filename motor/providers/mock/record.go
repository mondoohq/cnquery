package mock

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
)

func hashCmd(message string) string {
	hash := sha256.New()
	hash.Write([]byte(message))
	return hex.EncodeToString(hash.Sum(nil))
}

func NewRecordProvider(p providers.Transport) (*MockRecordProvider, error) {
	mock, err := New()
	if err != nil {
		return nil, err
	}

	recordWrapper := &MockRecordProvider{
		mock:    mock,
		observe: p,
	}

	// always run identifier here to collect the identifier that is only available via the provider
	// we do not care about the output here, we only want to make sure its being tracked
	recordWrapper.Identifier()

	return recordWrapper, nil
}

type MockRecordProvider struct {
	observe providers.Transport
	mock    *Provider
}

func (p *MockRecordProvider) Watched() providers.Transport {
	return p.observe
}

func (p *MockRecordProvider) Export() (*TomlData, error) {
	return Export(p.mock)
}

func (p *MockRecordProvider) ExportData() ([]byte, error) {
	return ExportData(p.mock)
}

func (p *MockRecordProvider) RunCommand(command string) (*providers.Command, error) {
	cmd, err := p.observe.RunCommand(command)
	if err != nil {
		// we do not record errors yet
		return nil, err
	}

	if cmd != nil {
		stdout := ""
		stderr := ""

		stdoutData, err := io.ReadAll(cmd.Stdout)
		if err == nil {
			stdout = string(stdoutData)
		}
		stderrData, err := io.ReadAll(cmd.Stderr)
		if err == nil {
			stderr = string(stderrData)
		}

		// store command
		p.mock.Commands[hashCmd(command)] = &Command{
			Command:    command,
			Stdout:     stdout,
			Stderr:     stderr,
			ExitStatus: cmd.ExitStatus,
		}
	}

	// read command from mock
	return p.mock.RunCommand(command)
}

func (p *MockRecordProvider) FS() afero.Fs {
	fs := p.observe.FS()
	return NewRecordFS(fs, p.mock.Fs)
}

func (p *MockRecordProvider) FileInfo(name string) (providers.FileInfoDetails, error) {
	enonet := false
	stat, err := p.observe.FileInfo(name)
	if err == os.ErrNotExist {
		enonet = true
	}

	fMock, ok := p.mock.Fs.Files[name]
	if !ok {
		fMock = &MockFileData{}
	}

	fMock.Path = name
	fMock.Enoent = enonet
	fMock.StatData = FileInfo{
		Mode: stat.Mode.FileMode,
		// TODO: add size if required
		// ModTime: stat.ModTime,
		// IsDir:   stat.IsDir,
		Uid: stat.Uid,
		Gid: stat.Gid,
	}

	p.mock.Fs.Files[name] = fMock

	return stat, err
}

func (p *MockRecordProvider) Capabilities() providers.Capabilities {
	caps := p.observe.Capabilities()
	p.mock.TransportInfo.Capabilities = caps
	return caps
}

func (p *MockRecordProvider) Close() {
	p.observe.Close()
}

func (p *MockRecordProvider) Kind() providers.Kind {
	k := p.observe.Kind()
	p.mock.TransportInfo.Kind = k
	return k
}

func (p *MockRecordProvider) Runtime() string {
	runtime := p.observe.Runtime()
	p.mock.TransportInfo.Runtime = runtime
	return runtime
}

func (p *MockRecordProvider) Identifier() (string, error) {
	identifiable, ok := p.observe.(providers.TransportPlatformIdentifier)
	if !ok {
		return "", errors.New("the transportid detector is not supported for transport")
	}

	id, err := identifiable.Identifier()
	if err == nil {
		p.mock.TransportInfo.ID = id
	}
	return id, err
}

func (p *MockRecordProvider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return p.observe.PlatformIdDetectors()
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
	// we need to check it here since toml does not allow to have empty names
	if name == "" {
		return nil, os.ErrNotExist
	}

	enonet := false
	content := []byte{}
	var fi FileInfo

	f, err := fs.observe.Open(name)
	if err == os.ErrNotExist {
		enonet = true
	} else if err != nil {
		return nil, err
	} else {
		// if recording is active, we also collect stats
		stat, err := f.Stat()
		if err == nil {
			fi = NewMockFileInfo(stat)
		} else {
			log.Warn().Err(err).Str("file", name).Msg("could not stat file for recording")
		}

		// only read the file content if the file is actually a file and not a directory
		if !fi.IsDir {
			data, err := ioutil.ReadAll(f)
			defer f.Close()
			if err != nil {
				return nil, err
			}
			content = data
		}
	}

	fMock, ok := fs.mock.Files[name]
	if !ok {
		fMock = &MockFileData{}
	}

	fMock.Data = content
	fMock.Path = name
	fMock.Enoent = enonet
	fMock.StatData = fi

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

func NewMockFileInfo(stat os.FileInfo) FileInfo {
	if stat == nil {
		return FileInfo{}
	}
	fi := FileInfo{
		Mode:    stat.Mode(),
		ModTime: stat.ModTime(),
		IsDir:   stat.IsDir(),
		// Uid:     0,
		// Gid:     0,
	}
	return fi
}

func (fs recordFS) Stat(name string) (os.FileInfo, error) {
	// we need to check it here since toml does not allow to have empty names
	if name == "" {
		return nil, os.ErrNotExist
	}

	enonet := false
	stat, err := fs.observe.Stat(name)
	if err == os.ErrNotExist {
		enonet = true
	}

	fMock, ok := fs.mock.Files[name]
	if !ok {
		fMock = &MockFileData{}
	}

	fMock.Path = name
	fMock.Enoent = enonet
	fMock.StatData = NewMockFileInfo(stat)
	fs.mock.Files[name] = fMock

	return stat, err
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

func (fs recordFS) Chown(name string, uid, gid int) error {
	return fs.observe.Chown(name, uid, gid)
}
