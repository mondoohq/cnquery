// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tmp

// This go package should be used across cnquery to generate temporary files
// and directories.

import (
	"os"

	"github.com/rs/zerolog/log"
)

// File creates a new temporary file.
//
// By default `File()` uses the default directory to create a new temporary file,
// to change the temporary directory use the environment variable `MONDOO_TMP_DIR`
func File() (*os.File, error) {
	tmpDir := ""
	if os.Getenv("MONDOO_TMP_DIR") != "" {
		tmpDir = os.Getenv("MONDOO_TMP_DIR")
		log.Debug().
			Str("custom_tmp_dir", tmpDir).
			Msg("creating temp file in custom temp directory")
	}
	return os.CreateTemp(tmpDir, "mondoo.tmp")
}
