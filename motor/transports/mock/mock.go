package mock

import (
	"bytes"
	"errors"
	"github.com/spf13/afero"

	"go.mondoo.io/mondoo/motor/transports"
)

type Command struct {
	Command    string `toml:"command"`
	Stdout     string `toml:"stdout"`
	Stderr     string `toml:"stderr"`
	ExitStatus int    `toml:"exit_status"`
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

// Transport holds the transport layer that runs on virtual data only
type Transport struct {
	Commands map[string]*Command
	Missing  map[string]map[string]bool
	Fs       *mockFS
	kind     transports.Kind
	runtime  string
}

// RunCommand returns the results of a command found in the nock registry
func (m *Transport) RunCommand(command string) (*transports.Command, error) {
	res := transports.Command{Command: command, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}

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

func (m *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	fs := m.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return transports.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)
	if stat, ok := stat.Sys().(*transports.FileInfo); ok {
		uid = int64(stat.Uid)
		gid = int64(stat.Gid)
	}

	mode := stat.Mode()

	return transports.FileInfoDetails{
		Mode: transports.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

// Close is used to terminate the connection, nothing for Transport
func (m *Transport) Close() {
	// no op
}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_RunCommand,
		transports.Capability_File,
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

func (t *Transport) Kind() transports.Kind {
	return t.kind
}

func (t *Transport) Runtime() string {
	return t.runtime
}
