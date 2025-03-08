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
	platformInfo := clouddetect.Detect(conn, conn.Asset().GetPlatform())
	if platformInfo == nil {
		log.Debug().Msg("os.cloud> unable to detect cloud")
		return &none{}, nil
	}

	log.Debug().Str("cloud", platformInfo.Name).Msg("os.cloud> detected")
	switch platformInfo.CloudProvider {
	case clouddetect.AWS:
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
