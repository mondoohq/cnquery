package provider

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared"
)

func (s *Service) detect(asset *inventory.Asset, conn shared.Connection) error {
	// TODO: handle all platforms for all the individual assets somehow

	connName := conn.Name()
	asset.Id = connName
	asset.Name = connName

	asset.Platform = conn.Platform()

	return nil
}
