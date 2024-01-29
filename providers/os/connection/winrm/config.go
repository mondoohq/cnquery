// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package winrm

import (
	"github.com/masterzen/winrm"
	"github.com/rs/zerolog/log"
	"os"
)

func DefaultConfig(endpoint *winrm.Endpoint) *winrm.Endpoint {
	// use default port if port is 0
	if endpoint.Port <= 0 {
		endpoint.Port = 5986
	}

	if endpoint.Port == 5985 {
		log.Warn().Msg("winrm port 5985 is using http communication instead of https, passwords are not encrypted")
		endpoint.HTTPS = false
	}

	if os.Getenv("WINRM_DISABLE_HTTPS") == "true" {
		log.Warn().Msg("WINRM_DISABLE_HTTPS is set, winrm is using http communication instead of https, passwords are not encrypted")
		endpoint.HTTPS = false
	}

	return endpoint
}
