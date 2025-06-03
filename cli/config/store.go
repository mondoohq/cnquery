// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// list of field keys to avoid writing to disk
var fieldKeysToOmit = []string{"force"}

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
