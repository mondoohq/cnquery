package mock

import (
	"bytes"
	"errors"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
)

var _ providers.Transport = (*Transport)(nil)

type Command struct {
	PlatformID string `toml:"platform_id"`
	Command    string `toml:"command"`
	Stdout     string `toml:"stdout"`
	Stderr     string `toml:"stderr"`
	ExitStatus int    `toml:"exit_status"`
}

type TransportInfo struct {
	ID           string                 `toml:"id"`
	Capabilities []providers.Capability `toml:"capabilities"`
	Kind         providers.Kind         `toml:"kind"`
	Runtime      string                 `toml:"runtime"`
}

// Transport holds the transport layer that runs on virtual data only
type Transport struct {
	TransportInfo TransportInfo
	Commands      map[string]*Command
	Missing       map[string]map[string]bool
	Fs            *mockFS
}

// New creates a new Transport.
func New() (*Transport, error) {
	mt := &Transport{
		Commands: make(map[string]*Command),
		Fs:       NewMockFS(),
	}

	mt.Missing = make(map[string]map[string]bool)
	mt.Missing["file"] = make(map[string]bool)
	mt.Missing["command"] = make(map[string]bool)
	return mt, nil
}

// RunCommand returns the results of a command found in the nock registry
func (m *Transport) RunCommand(command string) (*providers.Command, error) {
	res := providers.Command{Command: command, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}

	// we check both the command and the sha sum

	c, ok := m.Commands[command]
	if !ok {
		// try to fetch command by hash (more reliable for whitespace)
		c, ok = m.Commands[hashCmd(command)]
	}

	// handle case where the command was not found
	if !ok {
		res.Stdout.Write([]byte(""))
		res.Stderr.Write([]byte("command not found"))
		res.ExitStatus = 1
		m.Missing["command"][command] = true
		return &res, errors.New("command not found: " + command)
	}

	res.ExitStatus = c.ExitStatus
	res.Stdout.Write([]byte(c.Stdout))
	res.Stderr.Write([]byte(c.Stderr))
	return &res, nil
}

func (m *Transport) FS() afero.Fs {
	if m.Fs == nil {
		m.Fs = NewMockFS()
	}
	return m.Fs
}

func (m *Transport) FileInfo(path string) (providers.FileInfoDetails, error) {
	fs := m.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return providers.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)
	if stat, ok := stat.Sys().(*providers.FileInfo); ok {
		uid = int64(stat.Uid)
		gid = int64(stat.Gid)
	}

	mode := stat.Mode()

	return providers.FileInfoDetails{
		Mode: providers.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

// Close is used to terminate the connection, nothing for Transport
func (m *Transport) Close() {
	// no op
}

func (t *Transport) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_RunCommand,
		providers.Capability_File,
	}
}

// // TODO, support directory streaming
// func (mf *MockFile) Tar() (io.ReadCloser, error) {
// 	if mf.file.Enoent {
// 		return nil, errors.New("no such file or directory")
// 	}

// 	f := mf.file
// 	fReader := ioutil.NopCloser(strings.NewReader(string(f.Content)))

// 	stat, err := mf.Stat()
// 	if err != nil {
// 		return nil, errors.New("could not retrieve file stats")
// 	}

// 	// create a pipe
// 	tarReader, tarWriter := io.Pipe()

// 	// convert raw stream to tar stream
// 	go fsutil.StreamFileAsTar(mf.Name(), stat, fReader, tarWriter)

// 	// return the reader
// 	return tarReader, nil
// }

func (t *Transport) Kind() providers.Kind {
	return t.TransportInfo.Kind
}

func (t *Transport) Runtime() string {
	return t.TransportInfo.Runtime
}

func (t *Transport) PlatformIdDetectors() []providers.PlatformIdDetector {
	detectors := []providers.PlatformIdDetector{
		providers.HostnameDetector,
	}

	if t.TransportInfo.ID != "" {
		detectors = append(detectors, providers.TransportPlatformIdentifierDetector)
	}

	return detectors
}

func (t *Transport) Identifier() (string, error) {
	if t.TransportInfo.ID == "" {
		return "", errors.New("the transportid detector is not supported for transport")
	}
	return t.TransportInfo.ID, nil
}
