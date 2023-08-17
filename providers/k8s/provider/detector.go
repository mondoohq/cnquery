package provider

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared"
)

func (s *Service) detect(asset *inventory.Asset, conn shared.Connection) error {
	// TODO: handle all platforms for all the individual assets somehow

	assetId, err := conn.AssetId()
	if err != nil {
		return err
	}
	asset.Id = assetId
	asset.Name = conn.Name()
	asset.PlatformIds = []string{assetId}

	asset.Platform = conn.Platform()

	return nil
}
