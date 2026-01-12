// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/id/hetznercompute"
)

const HETZNER Provider = "Hetzner"

// hetzner implements the OSCloud interface for Hetzner Cloud
type hetzner struct {
	conn shared.Connection
}

func (h *hetzner) Provider() Provider {
	return HETZNER
}

func (h *hetzner) Instance() (*InstanceMetadata, error) {
	mdsvc, err := hetznercompute.Resolve(h.conn, h.conn.Asset().GetPlatform())
	if err != nil {
		log.Debug().Err(err).Msg("os.cloud.hetzner> failed to get metadata resolver")
		return nil, err
	}
	metadata, err := mdsvc.RawMetadata()
	if err != nil {
		log.Debug().Err(err).Msg("os.cloud.hetzner> failed to get raw metadata")
		return nil, err
	}
	if metadata == nil {
		log.Debug().Msg("os.cloud.hetzner> no metadata found")
		return nil, errors.New("no metadata")
	}

	instanceMd := &InstanceMetadata{Metadata: metadata}

	m, ok := metadata.(map[string]any)
	if !ok {
		return instanceMd, errors.New("unexpected raw metadata")
	}

	if value, ok := m["hostname"]; ok {
		if hostname, ok := value.(string); ok {
			instanceMd.PrivateHostname = hostname
		}
	}

	if value, ok := m["public-ipv4"]; ok {
		if ipv4, ok := value.(string); ok && ipv4 != "" {
			instanceMd.PublicIpv4 = []Ipv4Address{{IP: ipv4}}
		}
	}

	return instanceMd, nil
}
