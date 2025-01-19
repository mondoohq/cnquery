// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package fsutil

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

type WalkDirFunc func(fs afero.Fs, path string) error

func WalkGlob(fs afero.Fs, paths []string, fn WalkDirFunc) error {
	// we search through default system locations
	for _, pattern := range paths {
		log.Debug().Str("path", pattern).Msg("searching for files")
		m, err := afero.Glob(fs, pattern)
		if err != nil {
			log.Debug().Err(err).Str("path", pattern).Msg("could not search for files")
			return err
		}
		for _, walkPath := range m {
			err = fn(fs, walkPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
