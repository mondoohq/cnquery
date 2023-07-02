package provider

import (
	"errors"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/providers/os/detector"
	"go.mondoo.com/cnquery/providers/os/id/aws"
	"go.mondoo.com/cnquery/providers/os/id/azure"
	"go.mondoo.com/cnquery/providers/os/id/gcp"
	"go.mondoo.com/cnquery/providers/os/id/hostname"
)

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

	if id, ok := hostname.Hostname(conn, asset.Platform); ok {
		asset.PlatformIds = append(asset.PlatformIds, id)
	}

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

	return nil
}

func relatedIds2assets(ids []string) []*asset.Asset {
	res := make([]*asset.Asset, len(ids))
	for i := range ids {
		res[i] = &asset.Asset{Id: ids[i]}
	}
	return res
}
