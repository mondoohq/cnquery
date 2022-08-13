package mock

import (
	"bytes"
	"errors"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
)

var _ providers.Transport = (*Provider)(nil)

type Command struct {
	PlatformID string `toml:"platform_id"`
	Command    string `toml:"command"`
	Stdout     string `toml:"stdout"`
	Stderr     string `toml:"stderr"`
	ExitStatus int    `toml:"exit_status"`
}

type MockProviderInfo struct {
	ID           string                 `toml:"id"`
	Capabilities []providers.Capability `toml:"capabilities"`
	Kind         providers.Kind         `toml:"kind"`
	Runtime      string                 `toml:"runtime"`
}

// Provider holds the transport layer that runs on virtual data only
type Provider struct {
	TransportInfo MockProviderInfo
	Commands      map[string]*Command
	Missing       map[string]map[string]bool
	Fs            *mockFS
}

// New creates a new Provider.
func New() (*Provider, error) {
	mt := &Provider{
		Commands: make(map[string]*Command),
		Fs:       NewMockFS(),
	}

	mt.Missing = make(map[string]map[string]bool)
	mt.Missing["file"] = make(map[string]bool)
	mt.Missing["command"] = make(map[string]bool)
	return mt, nil
}

// RunCommand returns the results of a command found in the nock registry
func (p *Provider) RunCommand(command string) (*providers.Command, error) {
	res := providers.Command{Command: command, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}

	// we check both the command and the sha sum

	c, ok := p.Commands[command]
	if !ok {
		// try to fetch command by hash (more reliable for whitespace)
		c, ok = p.Commands[hashCmd(command)]
	}

	// handle case where the command was not found
	if !ok {
		res.Stdout.Write([]byte(""))
		res.Stderr.Write([]byte("command not found"))
		res.ExitStatus = 1
		p.Missing["command"][command] = true
		return &res, errors.New("command not found: " + command)
	}

	res.ExitStatus = c.ExitStatus
	res.Stdout.Write([]byte(c.Stdout))
	res.Stderr.Write([]byte(c.Stderr))
	return &res, nil
}

func (p *Provider) FS() afero.Fs {
	if p.Fs == nil {
		p.Fs = NewMockFS()
	}
	return p.Fs
}

func (p *Provider) FileInfo(path string) (providers.FileInfoDetails, error) {
	fs := p.FS()
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

// Close is used to terminate the connection, nothing for Provider
func (p *Provider) Close() {
	// no op
}

func (p *Provider) Capabilities() providers.Capabilities {
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

func (p *Provider) Kind() providers.Kind {
	return p.TransportInfo.Kind
}

func (p *Provider) Runtime() string {
	return p.TransportInfo.Runtime
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	detectors := []providers.PlatformIdDetector{
		providers.HostnameDetector,
	}

	if p.TransportInfo.ID != "" {
		detectors = append(detectors, providers.TransportPlatformIdentifierDetector)
	}

	return detectors
}

func (p *Provider) Identifier() (string, error) {
	if p.TransportInfo.ID == "" {
		return "", errors.New("the transportid detector is not supported for transport")
	}
	return p.TransportInfo.ID, nil
}
