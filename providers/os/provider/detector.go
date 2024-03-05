// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"slices"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/detector"
	"go.mondoo.com/cnquery/v10/providers/os/id/aws"
	"go.mondoo.com/cnquery/v10/providers/os/id/azure"
	"go.mondoo.com/cnquery/v10/providers/os/id/gcp"
	"go.mondoo.com/cnquery/v10/providers/os/id/hostname"
	"go.mondoo.com/cnquery/v10/providers/os/id/ids"
	"go.mondoo.com/cnquery/v10/providers/os/id/machineid"
	"go.mondoo.com/cnquery/v10/providers/os/id/sshhostkey"
)

// default id detectors
var IdDetectors = []string{
	ids.IdDetector_Hostname,
	ids.IdDetector_CloudDetect,
	ids.IdDetector_SshHostkey,
}

func hasDetector(detectors map[string]struct{}, any ...string) bool {
	for i := range any {
		if _, ok := detectors[any[i]]; ok {
			return true
		}
	}
	return false
}

func mapDetectors(raw []string) map[string]struct{} {
	if len(raw) == 0 {
		raw = IdDetectors
	}
	res := make(map[string]struct{}, len(raw))
	for _, v := range raw {
		res[v] = struct{}{}
	}
	return res
}

func (s *Service) detect(asset *inventory.Asset, conn shared.Connection) error {
	var ok bool
	asset.Platform, ok = detector.DetectOS(conn)
	if !ok {
		return errors.New("failed to detect OS")
	}
	if asset.Platform.Kind == "" {
		asset.Platform.Kind = "baremetal"
	}
	if asset.Connections[0].Runtime == "vagrant" {
		// detect overrides this
		asset.Platform.Kind = "virtualmachine"
	}

	var detectors map[string]struct{}
	if !slices.Contains([]string{"container-image", "container"}, asset.Platform.Kind) {
		detectors = mapDetectors(asset.IdDetector)
	}

	if hasDetector(detectors, ids.IdDetector_Hostname) {
		log.Debug().Msg("run hostname id detector")
		if id, ok := hostname.Hostname(conn, asset.Platform); ok {
			asset.PlatformIds = append(asset.PlatformIds, id)
		}
	}

	if hasDetector(detectors, ids.IdDetector_CloudDetect) {
		log.Debug().Msg("run cloud platform detector")
		if id, name, related := aws.Detect(conn, asset.Platform); id != "" {
			asset.PlatformIds = append(asset.PlatformIds, id)
			asset.Name = name
			asset.RelatedAssets = append(asset.RelatedAssets, relatedIds2assets(related)...)
		}

		if id, _, related := azure.Detect(conn, asset.Platform); id != "" {
			asset.PlatformIds = append(asset.PlatformIds, id)
			asset.RelatedAssets = append(asset.RelatedAssets, relatedIds2assets(related)...)
		}

		if id, _, related := gcp.Detect(conn, asset.Platform); id != "" {
			asset.PlatformIds = append(asset.PlatformIds, id)
			asset.RelatedAssets = append(asset.RelatedAssets, relatedIds2assets(related)...)
		}
	}

	if hasDetector(detectors, ids.IdDetector_SshHostkey) {
		log.Debug().Msg("run ssh id detector")
		ids, err := sshhostkey.Detect(conn, asset.Platform)
		if err != nil {
			log.Warn().Err(err).Msg("failure in ssh hostkey detector")
		} else {
			asset.PlatformIds = append(asset.PlatformIds, ids...)
		}
	}

	if hasDetector(detectors, ids.IdDetector_MachineID) {
		log.Debug().Msg("run machineID id detector")
		id, hostErr := machineid.MachineId(conn, asset.Platform)
		if hostErr != nil {
			log.Warn().Err(hostErr).Msg("failure in machineID detector")
		} else if id != "" {
			asset.PlatformIds = append(asset.PlatformIds, id)
		}
	}

	// First sort the platform IDs and then call Compact, because Compact removes only consecutive duplicates
	slices.Sort(asset.PlatformIds)
	asset.PlatformIds = slices.Compact(asset.PlatformIds)

	// If the asset connection had the SkipDiscovery flag and the current asset doesn't, we just performed
	// discovery for the asset and we need to update it.
	if conn.Asset().Connections[0].SkipDiscovery && !asset.Connections[0].SkipDiscovery {
		conn.UpdateAsset(asset)
	}
	return nil
}

func relatedIds2assets(ids []string) []*inventory.Asset {
	res := make([]*inventory.Asset, len(ids))
	for i := range ids {
		res[i] = &inventory.Asset{Id: ids[i]}
	}
	return res
}
