// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/detector"
	"go.mondoo.com/cnquery/providers/os/id/aws"
	"go.mondoo.com/cnquery/providers/os/id/azure"
	"go.mondoo.com/cnquery/providers/os/id/gcp"
	"go.mondoo.com/cnquery/providers/os/id/hostname"
	"go.mondoo.com/cnquery/providers/os/id/machineid"
	"go.mondoo.com/cnquery/providers/os/id/sshhostkey"
)

const (
	IdDetector_Hostname    = "hostname"
	IdDetector_MachineID   = "machine-id"
	IdDetector_CloudDetect = "cloud-detect"
	IdDetector_SshHostkey  = "ssh-host-key"

	// FIXME: DEPRECATED, remove in v9.0 vv
	// this is now cloud-detect
	IdDetector_AwsEc2 = "aws-ec2"
	// ^^

	// IdDetector_PlatformID = "transport-platform-id" // TODO: how does this work?
)

var IdDetectors = []string{
	IdDetector_Hostname,
	IdDetector_MachineID,
	IdDetector_CloudDetect,
	IdDetector_SshHostkey,
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

	var detectors map[string]struct{}
	if asset.Platform.Kind != "container-image" {
		detectors = mapDetectors(asset.IdDetector)
	}

	if hasDetector(detectors, IdDetector_Hostname) {
		if id, ok := hostname.Hostname(conn, asset.Platform); ok {
			asset.PlatformIds = append(asset.PlatformIds, id)
		}
	}

	if hasDetector(detectors, IdDetector_CloudDetect, IdDetector_AwsEc2) {
		if id, name, related := aws.Detect(conn, asset.Platform); id != "" {
			asset.PlatformIds = append(asset.PlatformIds, id)
			asset.Platform.Name = name
			asset.RelatedAssets = append(asset.RelatedAssets, relatedIds2assets(related)...)
		}

		if id, name, related := azure.Detect(conn, asset.Platform); id != "" {
			asset.PlatformIds = append(asset.PlatformIds, id)
			asset.Platform.Name = name
			asset.RelatedAssets = append(asset.RelatedAssets, relatedIds2assets(related)...)
		}

		if id, name, related := gcp.Detect(conn, asset.Platform); id != "" {
			asset.PlatformIds = append(asset.PlatformIds, id)
			asset.Platform.Name = name
			asset.RelatedAssets = append(asset.RelatedAssets, relatedIds2assets(related)...)
		}
	}

	if hasDetector(detectors, IdDetector_SshHostkey) {
		ids, err := sshhostkey.Detect(conn, asset.Platform)
		if err != nil {
			log.Warn().Err(err).Msg("failure in ssh hostkey detector")
		} else {
			asset.PlatformIds = append(asset.PlatformIds, ids...)
		}
	}

	if hasDetector(detectors, IdDetector_MachineID) {
		id, hostErr := machineid.MachineId(conn, asset.Platform)
		if hostErr != nil {
			log.Warn().Err(hostErr).Msg("failure in machineID detector")
		} else if id != "" {
			asset.PlatformIds = append(asset.PlatformIds, id)
		}
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
