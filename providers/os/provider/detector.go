package provider

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
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

func hasDetector(all []string, any ...string) bool {
	if len(all) == 0 {
		return true
	}
	for i := range any {
		if all[0] == any[i] {
			return true
		}
	}
	return false
}

func (s *Service) detect(asset *asset.Asset) error {
	conn, err := s.connect(asset)
	if err != nil {
		return err
	}

	var ok bool
	asset.Platform, ok = detector.DetectOS(conn)
	if !ok {
		return errors.New("failed to detect OS")
	}

	if hasDetector(asset.IdDetector, IdDetector_Hostname) {
		if id, ok := hostname.Hostname(conn, asset.Platform); ok {
			asset.PlatformIds = append(asset.PlatformIds, id)
		}
	}

	if hasDetector(asset.IdDetector, IdDetector_CloudDetect, IdDetector_AwsEc2) {
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

	if hasDetector(asset.IdDetector, IdDetector_SshHostkey) {
		ids, err := sshhostkey.Detect(conn, asset.Platform)
		if err != nil {
			log.Warn().Err(err).Msg("failure in ssh hostkey detector")
		} else {
			asset.PlatformIds = append(asset.PlatformIds, ids...)
		}
	}

	if hasDetector(asset.IdDetector, IdDetector_MachineID) {
		id, hostErr := machineid.MachineId(conn, asset.Platform)
		if hostErr != nil {
			log.Warn().Err(err).Msg("failure in machineID detector")
		} else if id != "" {
			asset.PlatformIds = append(asset.PlatformIds, id)
		}
	}

	return nil
}

func relatedIds2assets(ids []string) []*asset.Asset {
	res := make([]*asset.Asset, len(ids))
	for i := range ids {
		res[i] = &asset.Asset{Id: ids[i]}
	}
	return res
}
