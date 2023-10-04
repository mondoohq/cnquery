package connection

import (
	"os"

	"github.com/ctreminiom/go-atlassian/admin"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
)

type AtlassianConnection struct {
	id    uint32
	Conf  *inventory.Config
	asset *inventory.Asset
	admin *admin.Client
	// Add custom connection fields here
}

func NewAtlassianConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AtlassianConnection, error) {
	apiKey := os.Getenv("ATLASSIAN_KEY")
	admin, err := admin.New(nil)
	if err != nil {
		log.Fatal().Err(err)
	}
	admin.Auth.SetBearerToken(apiKey)
	admin.Auth.SetUserAgent("curl/7.54.0")

	conn := &AtlassianConnection{
		Conf:  conf,
		id:    id,
		asset: asset,
		admin: admin,
	}

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

func (c *AtlassianConnection) Admin() *admin.Client {
	return c.admin
}
