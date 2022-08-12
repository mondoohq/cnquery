package mock

import (
	"bytes"
	"errors"
	"io/ioutil"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/providers"
)

// Data holds the mocked data entries
type TomlData struct {
	TransportInfo TransportInfo            `toml:"transport_info"`
	Commands      map[string]*Command      `toml:"commands"`
	Files         map[string]*MockFileData `toml:"files"`
}

func Parse(data string) (*TomlData, error) {
	tomlContent := &TomlData{}
	if _, err := toml.Decode(string(data), &tomlContent); err != nil {
		return nil, errors.New("could not decode toml: " + err.Error())
	}

	// do data sanitization
	for path, f := range tomlContent.Files {
		f.Path = path
	}

	log.Debug().Int("commands", len(tomlContent.Commands)).Int("files", len(tomlContent.Files)).Msg("mock> loaded data successfully")

	// trace information
	for k := range tomlContent.Commands {
		log.Trace().Str("cmd", k).Msg("load command")
	}

	for k := range tomlContent.Files {
		log.Trace().Str("file", k).Msg("load file")
	}

	return tomlContent, nil
}

func LoadFile(mock *Transport, path string) error {
	log.Debug().Str("path", path).Msg("mock> load toml into mock backend")

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.New("could not open: " + path)
	}

	return Load(mock, string(data))
}

func Load(mock *Transport, data string) error {
	tomlData, err := Parse(data)
	if err != nil {
		return err
	}

	// copy references
	mock.Commands = tomlData.Commands
	mock.Fs.Files = tomlData.Files
	mock.TransportInfo = tomlData.TransportInfo
	return nil
}

// Export returns a struct that can be used to export toml
func Export(mock *Transport) (*TomlData, error) {
	tomlData := &TomlData{}
	tomlData.Commands = mock.Commands
	tomlData.Files = mock.Fs.Files
	tomlData.TransportInfo = mock.TransportInfo
	return tomlData, nil
}

func ExportData(mock *Transport) ([]byte, error) {
	data, err := Export(mock)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	e := toml.NewEncoder(&buf)
	err = e.Encode(data)
	return buf.Bytes(), err
}

// New returns a mock backend and loads the toml file by default
func NewFromToml(tc *providers.TransportConfig) (*Transport, error) {
	if tc.Options == nil || tc.Options["path"] == "" {
		return nil, errors.New("path is required")
	}

	path := tc.Options["path"]

	transport, err := New()
	if err != nil {
		return nil, err
	}

	err = LoadFile(transport, path)
	if err != nil {
		log.Error().Err(err).Str("toml", path).Msg("mock> could not load toml data")
		return nil, err
	}

	if tc.Options["hostname"] != "" {
		transport.Commands[hashCmd("hostname")] = &Command{
			Stdout: tc.Options["hostname"],
		}
	}

	return transport, nil
}

// NewTestMockTransport is a sugar method to simplify writing tests with the mock backend
func NewFromTomlFile(filepath string) (*Transport, error) {
	return NewFromToml(&providers.TransportConfig{
		Backend: providers.ProviderType_MOCK,
		Options: map[string]string{
			"path": filepath,
		},
	})
}
