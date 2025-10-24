package connection

import (
	"errors"
	"os"
	"strings"

	vt "github.com/VirusTotal/vt-go"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/vault"
)

const (
	DiscoveryNone = "none"
)

type VirustotalConnection struct {
	plugin.Connection
	Conf   *inventory.Config
	asset  *inventory.Asset
	client *vt.Client
}

func NewVirustotalConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*VirustotalConnection, error) {
	conn := &VirustotalConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	apiKey := firstNonEmpty(
		os.Getenv("VIRUSTOTAL_API_KEY"),
		os.Getenv("VT_API_KEY"),
	)

	if len(conf.Credentials) > 0 {
		for i := range conf.Credentials {
			cred := conf.Credentials[i]
			switch cred.Type {
			case vault.CredentialType_password:
				apiKey = string(cred.Secret)
			default:
				log.Warn().
					Str("credential-type", cred.Type.String()).
					Msg("unsupported credential type for VirusTotal provider")
			}
		}
	}

	if apiKey == "" {
		return nil, errors.New("a VirusTotal API key is required, pass --api-key '<key>' or set VT_API_KEY")
	}

	client := vt.NewClient(apiKey)
	client.Agent = "cnquery"

	conn.client = client
	return conn, nil
}

func (c *VirustotalConnection) Name() string {
	return "virustotal"
}

func (c *VirustotalConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *VirustotalConnection) Client() *vt.Client {
	return c.client
}

func (c *VirustotalConnection) PlatformInfo() (*inventory.Platform, error) {
	return &inventory.Platform{
		Name:                  "virustotal",
		Title:                 "VirusTotal",
		Family:                []string{"virustotal"},
		Kind:                  "api",
		Runtime:               "virustotal",
		TechnologyUrlSegments: []string{"threat-intel", "virustotal", "profile"},
	}, nil
}

func (c *VirustotalConnection) Identifier() string {
	base := "//platformid.api.mondoo.app/runtime/virustotal"
	if len(c.asset.Connections) == 0 {
		return base
	}

	conf := c.asset.Connections[0]
	if conf.Type != "" {
		return base + "/" + strings.ToLower(conf.Type)
	}
	return base
}

func firstNonEmpty(values ...string) string {
	for i := range values {
		if values[i] != "" {
			return values[i]
		}
	}
	return ""
}
