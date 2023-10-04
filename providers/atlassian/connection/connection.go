package connection

import (
	"os"

	"github.com/ctreminiom/go-atlassian/admin"
	"github.com/ctreminiom/go-atlassian/confluence"
	"github.com/ctreminiom/go-atlassian/jira/v2"
	_ "github.com/ctreminiom/go-atlassian/jira/v3"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
)

type AtlassianConnection struct {
	id        uint32
	Conf      *inventory.Config
	asset     *inventory.Asset
	admin     *admin.Client
	jira      *v2.Client
	confluece *confluence.Client
	// Add custom connection fields here
}

func NewAtlassianConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AtlassianConnection, error) {
	apiKey := os.Getenv("ATLASSIAN_KEY")
	token := os.Getenv("ATLASSIAN_TOKEN")
	host := "https://lunalectric.atlassian.net"
	mail := "marius@mondoo.com"
	admin, err := admin.New(nil)
	if err != nil {
		log.Fatal().Err(err)
	}
	admin.Auth.SetBearerToken(apiKey)
	admin.Auth.SetUserAgent("curl/7.54.0")

	jira, err := v2.New(nil, host)
	if err != nil {
		log.Fatal().Err(err)
	}

	jira.Auth.SetBasicAuth(mail, token)
	jira.Auth.SetUserAgent("curl/7.54.0")

	confluence, err := confluence.New(nil, host)
	if err != nil {
		log.Fatal().Err(err)
	}

	confluence.Auth.SetBasicAuth(mail, token)
	confluence.Auth.SetUserAgent("curl/7.54.0")

	conn := &AtlassianConnection{
		Conf:      conf,
		id:        id,
		asset:     asset,
		admin:     admin,
		jira:      jira,
		confluece: confluence,
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

func (c *AtlassianConnection) Jira() *v2.Client {
	return c.jira
}

func (c *AtlassianConnection) Confluence() *confluence.Client {
	return c.confluece
}
