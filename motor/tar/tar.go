package tar

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"go.mondoo.io/mondoo/motor/motorutil"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/types"
)

type TarFile struct {
	transport *Transport
	path      string
	header    *tar.Header
}

func (f *TarFile) Name() string {
	return f.path
}

func (f *TarFile) Open() (types.FileStream, error) {
	return f.transport.open(f.header)
}

func (f *TarFile) Stat() (os.FileInfo, error) {
	return f.transport.stat(f.header)
}

func (f *TarFile) Tar() (io.ReadCloser, error) {
	return f.transport.tar(f.path, f.header)
}

func (f *TarFile) Exists() bool {
	_, err := f.Stat()
	if err != nil {
		return false
	}
	return true
}

func (f *TarFile) Readdir(n int) ([]os.FileInfo, error) {
	return nil, errors.New("not implemented yet")
}

func (f *TarFile) Readdirnames(n int) ([]string, error) {
	return nil, errors.New("not implemented yet")
}

func New(endpoint *types.Endpoint) (*Transport, error) {
	return NewWithClose(endpoint, nil)
}

type closefn func()

func NewWithClose(endpoint *types.Endpoint, close closefn) (*Transport, error) {
	t := &Transport{
		Source:  endpoint.Path,
		FileMap: make(map[string]*tar.Header),
		CloseFN: close,
	}

	var err error
	if endpoint != nil && len(endpoint.Path) > 0 {
		err := t.LoaTarFile(endpoint.Path)
		if err != nil {
			log.Error().Err(err).Str("tar", endpoint.Path).Msg("tar> could not load tar file")
			return nil, err
		}
	}
	return t, err
}

// Transport loads tar files and make them available
type Transport struct {
	Source  string
	FileMap map[string]*tar.Header
	CloseFN closefn
}

func (m *Transport) RunCommand(command string) (*types.Command, error) {
	res := types.Command{Command: command, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, ExitStatus: -1}
	return &res, nil
}

func (m *Transport) File(path string) (types.File, error) {
	h, ok := m.FileMap[path]
	if !ok {
		return nil, errors.New("file does not exist")
	}
	return &TarFile{path: path, header: h, transport: m}, nil
}

func (m *Transport) Close() {
	if m.CloseFN != nil {
		m.CloseFN()
	}

}

func (m *Transport) stat(header *tar.Header) (os.FileInfo, error) {
	statHeader := header
	if header.Typeflag == tar.TypeSymlink {
		path := m.resolveSymlink(header)
		h, ok := m.FileMap[m.Abs(path)]
		if !ok {
			return nil, errors.New("could not find " + path)
		}
		statHeader = h
	}
	return statHeader.FileInfo(), nil
}

func (m *Transport) open(header *tar.Header) (types.FileStream, error) {
	log.Debug().Str("file", header.Name).Msg("tar> load file content")

	// open tar file
	f, err := os.Open(m.Source)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	path := header.Name
	if header.Typeflag == tar.TypeSymlink {
		path = m.resolveSymlink(header)
	}

	// extract file from tar stream
	reader, err := motorutil.ExtractFileFromTarStream(path, f)
	if err != nil {
		return nil, err
	}
	return ioutil.NopCloser(reader), nil
}

// resolve symlink file
func (m *Transport) resolveSymlink(header *tar.Header) string {
	dest := header.Name
	source := header.Linkname
	path := filepath.Clean(filepath.Join(dest, "..", source))
	log.Debug().Str("link", header.Linkname).Str("header", header.Name).Str("path", path).Msg("tar> is symlink")
	return path
}

func (m *Transport) tar(path string, header *tar.Header) (types.FileStream, error) {
	fReader, err := m.open(header)
	if err != nil {
		return nil, err
	}

	// create a pipe
	tarReader, tarWriter := io.Pipe()

	// get file info, header my just include symlink fileinfo
	fi, err := m.stat(header)
	if err != nil {
		return nil, err
	}

	// convert raw stream to tar stream
	go motorutil.StreamFileAsTar(header.Name, fi, fReader, tarWriter)

	// return the reader
	return tarReader, nil
}

// docker images only use relative paths, we need to make them absolute here
func (m *Transport) Abs(path string) string {
	return filepath.Join("/", path)
}

func (m *Transport) LoadTar(stream io.Reader) error {
	tr := tar.NewReader(stream)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Err(err).Msg("tar> error reading tar stream")
			return err
		}

		path := m.Abs(h.Name)
		m.FileMap[path] = h
	}
	log.Debug().Int("files", len(m.FileMap)).Msg("tar> successfully loaded")
	return nil
}

func (m *Transport) LoaTarFile(path string) error {
	log.Debug().Str("path", path).Msg("tar> load tar file into backend")

	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return err
	}

	return m.LoadTar(f)
}
