package mock

import (
	"bytes"
	"errors"
	"os"

	"github.com/spf13/afero"

	"go.mondoo.io/mondoo/motor/motoros/capabilities"
	"go.mondoo.io/mondoo/motor/motoros/types"
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
}

// RunCommand returns the results of a command found in the nock registry
func (m *Transport) RunCommand(command string) (*types.Command, error) {
	res := types.Command{Command: command, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}

	c, ok := m.Commands[command]
	if !ok {
		res.Stdout.Write([]byte(""))
		res.Stderr.Write([]byte("command not found"))
		m.Missing["command"][command] = true
		return &res, errors.New("command not found: " + command)
	}

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

func (m *Transport) File(path string) (afero.File, error) {
	f, err := m.FS().Open(path)
	if err == os.ErrNotExist {
		m.Missing["file"][path] = true
	}
	return f, err
}

// Close is used to terminate the connection, nothing for Transport
func (m *Transport) Close() {
	// no op
}

func (t *Transport) Capabilities() []capabilities.Capability {
	return []capabilities.Capability{
		capabilities.RunCommand,
		capabilities.File,
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
