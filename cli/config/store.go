// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

// list of field keys to avoid writing to disk
var fieldKeysToOmit = []string{"force"}

// MarshalConfig serializes cfg using the serialization format implied by the
// config file path's extension: a ".json" path produces JSON, everything else
// (".yaml", ".yml", or no extension) produces YAML. This mirrors viper's own
// extension-based format detection (used by WriteConfig) so a config file
// loaded as JSON is written back as JSON rather than silently converted to
// YAML.
//
// Both formats key off the struct's json tags (sigs.k8s.io/yaml marshals via
// JSON under the hood), so the on-disk key names are identical regardless of
// format.
func MarshalConfig(path string, cfg *Config) ([]byte, error) {
	if strings.EqualFold(filepath.Ext(path), ".json") {
		return json.MarshalIndent(cfg, "", "  ")
	}
	return yaml.Marshal(cfg)
}

func StoreConfig() error {
	path := viper.ConfigFileUsed()
	log.Info().Str("path", path).Msg("saving config")

	// create new file if it does not exist
	osFs := afero.NewOsFs()
	if _, err := osFs.Stat(path); os.IsNotExist(err) {
		log.Info().Str("path", path).Msg("config file does not exist, create a new one")
		// create the directory if it does not exist
		err = osFs.MkdirAll(filepath.Dir(path), 0o755)
		if err != nil {
			return errors.Wrap(err, "failed to save mondoo config")
		}

		// write file
		err = os.WriteFile(path, []byte{}, 0o644)
		if err != nil {
			return errors.Wrap(err, "failed to save mondoo config")
		}
	} else if err != nil {
		return errors.Wrap(err, "failed to check stats for mondoo config")
	}

	// omit fields before storing the configuration
	for _, field := range fieldKeysToOmit {
		viper.Set(field, nil)
	}

	return viper.WriteConfig()
}
