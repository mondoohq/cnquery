package mock

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/types"
)

type Command struct {
	Command    string `toml:"command"`
	Stdout     string `toml:"stdout"`
	Stderr     string `toml:"stderr"`
	ExitStatus int    `toml:"exit_status"`
}

type FileInfo struct {
	Mode    os.FileMode `toml:"mode"`
	ModTime time.Time   `toml:"time"`
	IsDir   bool        `toml:"isdir"`
}

type File struct {
	Path    string   `toml:"path"`
	Content string   `toml:"content"`
	Stat    FileInfo `toml:"stat"`
	Enoent  bool     `toml:"enoent"`
}

// New creates a new Transport.
func New() (*Transport, error) {
	mt := &Transport{
		Commands: make(map[string]*Command),
		Files:    make(map[string]*File),
	}

	mt.Missing = make(map[string]map[string]bool)
	mt.Missing["file"] = make(map[string]bool)
	mt.Missing["command"] = make(map[string]bool)
	return mt, nil
}

// Transport holds the transport layer that runs on virtual data only
type Transport struct {
	Commands map[string]*Command
	Files    map[string]*File
	Missing  map[string]map[string]bool
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

// File will return the given path with a mocked file
func (m *Transport) File(path string) (types.File, error) {
	f, ok := m.Files[path]
	if !ok || f.Enoent {
		m.Missing["file"][path] = true
		return nil, errors.New("no such file or directory")
	}
	return &MockFile{file: f}, nil
}

// Close is used to terminate the connection, nothing for Transport
func (m *Transport) Close() {
	// no op
}

// load files from a tar stream
func (m *Transport) LoadFromTarStream(stream io.Reader) error {
	tr := tar.NewReader(stream)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Err(err).Msg("error reading tar stream")
			return err
		}

		content, err := ioutil.ReadAll(tr)
		if err != nil {
			log.Error().Str("file", h.Name).Err(err).Msg("mock> could not load file data")
		} else {
			log.Debug().Str("file", h.Name).Str("content", string(content)).Msg("mock> content")
		}
		fi := h.FileInfo()
		m.Files[h.Name] = &File{
			Path:    h.Name,
			Content: string(content),
			Stat: FileInfo{
				Mode:    fi.Mode(),
				IsDir:   fi.IsDir(),
				ModTime: fi.ModTime(),
			},
		}
		log.Debug().Str("file", h.Name).Msg("mock> add file to mock backend")
	}
	return nil
}
