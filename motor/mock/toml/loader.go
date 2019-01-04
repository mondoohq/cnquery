package toml

import (
	"errors"
	"io/ioutil"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/mock"
	"go.mondoo.io/mondoo/motor/types"
)

// Data holds the mocked data entries
type TomlData struct {
	Commands map[string]*mock.Command `toml:"commands"`
	Files    map[string]*mock.File    `toml:"files"`
}

func ParseToml(data string) (*TomlData, error) {
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

func LoadToml(mock *mock.Transport, data string) error {
	tomlData, err := ParseToml(data)
	if err != nil {
		return err
	}

	// copy references
	mock.Commands = tomlData.Commands
	mock.Files = tomlData.Files
	return nil
}

func LoadTomlFile(mock *mock.Transport, path string) error {
	log.Debug().Str("path", path).Msg("mock> load toml into mock backend")

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.New("could not open: " + path)
	}

	return LoadToml(mock, string(data))
}

// ExportToToml returns a struct that can be used to export toml
func ExportToToml(mock *mock.Transport) (*TomlData, error) {
	tomlData := &TomlData{}
	tomlData.Commands = mock.Commands
	tomlData.Files = mock.Files
	return tomlData, nil
}

// New returns a mock backend and loads the toml file by default
func New(endpoint *types.Endpoint) (types.Transport, error) {
	transport, err := mock.New()
	if err != nil {
		return nil, err
	}

	if endpoint != nil && len(endpoint.Path) > 0 {
		err := LoadTomlFile(transport, endpoint.Path)
		if err != nil {
			log.Error().Err(err).Str("toml", endpoint.Path).Msg("mock> could not load toml data")
			return nil, err
		}
	}

	return transport, nil
}
