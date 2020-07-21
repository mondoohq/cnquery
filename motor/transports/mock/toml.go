package mock

import (
	"bytes"
	"errors"
	"io/ioutil"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/motorapi"
)

// Data holds the mocked data entries
type TomlData struct {
	Commands map[string]*Command      `toml:"commands"`
	Files    map[string]*MockFileData `toml:"files"`
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

	return tomlContent, nil
}

func Load(mock *Transport, data string) error {
	tomlData, err := Parse(data)
	if err != nil {
		return err
	}

	// copy references
	mock.Commands = tomlData.Commands
	mock.Fs.Files = tomlData.Files
	return nil
}

func LoadFile(mock *Transport, path string) error {
	log.Debug().Str("path", path).Msg("mock> load toml into mock backend")

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.New("could not open: " + path)
	}

	return Load(mock, string(data))
}

// Export returns a struct that can be used to export toml
func Export(mock *Transport) (*TomlData, error) {
	tomlData := &TomlData{}
	tomlData.Commands = mock.Commands
	tomlData.Files = mock.Fs.Files
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
func NewFromToml(endpoint *motorapi.Endpoint) (*Transport, error) {
	transport, err := New()
	if err != nil {
		return nil, err
	}

	if endpoint != nil && len(endpoint.Path) > 0 {
		err := LoadFile(transport, endpoint.Path)
		if err != nil {
			log.Error().Err(err).Str("toml", endpoint.Path).Msg("mock> could not load toml data")
			return nil, err
		}
	}

	return transport, nil
}
