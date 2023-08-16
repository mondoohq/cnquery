package provider

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/gcp/connection"
)

func (s *Service) detect(asset *inventory.Asset, conn *connection.Connection) error {
	// TODO: handle all platforms for all the individual assets somehow

	connName := conn.Name()
	asset.Id = connName
	asset.Name = connName

	asset.Platform = &inventory.Platform{
		Name:    "GCP project", // TODO: add project name for project
		Family:  []string{"google"},
		Kind:    "gcp-object",
		Runtime: "gcp",
		Title:   "GCP Project",
	}

	return nil
}
