// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/id/clouddetect"
)

type Provider string

type OSCloud interface {
	Provider() Provider
	Instance() (*InstanceMetadata, error)
}

func Resolve(conn shared.Connection) (OSCloud, error) {
	// TODO fix cloud detect to return the type of cloud
	// Depends on https://github.com/mondoohq/cnquery/pull/5267
	identifier, _, _ := clouddetect.Detect(conn, conn.Asset().GetPlatform())
	log.Debug().Str("identified", identifier).Msg("os.cloud> cloud detected")
	var detectedCloud = ""
	if identifier != "" {
		detectedCloud = "aws"
	}
	// TODO fix ^^

	switch detectedCloud {
	case "aws":
		return &aws{conn}, nil
	default:
		return &none{}, nil
	}
}

const UNKNOWN Provider = "unknown"

type none struct{}

func (n *none) Provider() Provider {
	return UNKNOWN
}
func (n *none) Instance() (*InstanceMetadata, error) {
	return nil, errors.New("unknown provider information")
}
