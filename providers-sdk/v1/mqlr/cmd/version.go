// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/Masterminds/semver"
	"github.com/rs/zerolog/log"
)

var versionRegexp = regexp.MustCompile(`Version:\s*"([^"]+)"`)

// detectProviderVersion reads the provider's config/config.go relative to the
// given .lr file and returns the current version incremented by one patch level.
// If detection fails for any reason it logs a warning and returns defaultVersionField.
func detectProviderVersion(lrFile string) string {
	absLR, err := filepath.Abs(lrFile)
	if err != nil {
		log.Warn().Err(err).Msg("could not resolve lr file path, using default version")
		return defaultVersionField
	}

	// .lr files live in providers/<name>/resources/; walk up to provider root.
	resourcesDir := filepath.Dir(absLR)
	providerRoot := filepath.Dir(resourcesDir)
	configFile := filepath.Join(providerRoot, "config", "config.go")

	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Warn().Str("path", configFile).Msg("provider config not found, using default version")
		return defaultVersionField
	}

	matches := versionRegexp.FindSubmatch(data)
	if len(matches) < 2 {
		log.Warn().Str("path", configFile).Msg("could not extract version from config, using default version")
		return defaultVersionField
	}

	v, err := semver.NewVersion(string(matches[1]))
	if err != nil {
		log.Warn().Err(err).Str("version", string(matches[1])).Msg("could not parse provider version, using default version")
		return defaultVersionField
	}

	next := v.IncPatch()
	return next.String()
}
