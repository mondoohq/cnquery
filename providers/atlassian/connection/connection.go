package connection

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
)

type AtlassianConnection struct {
	id       uint32
	Conf     *inventory.Config
	asset    *inventory.Asset
	// Add custom connection fields here
}

func NewAtlassianConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AtlassianConnection, error) {
	conn := &AtlassianConnection{
		Conf:  conf,
		id:    id,
		asset: asset,
	}

	// initialize your connection here

	return conn, nil
}

func (c *AtlassianConnection) Name() string {
	return "atlassian"
}

func (c *AtlassianConnection) ID() uint32 {
	return c.id
}

func (c *AtlassianConnection) Asset() *inventory.Asset {
	return c.asset
}

