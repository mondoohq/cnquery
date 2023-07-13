package slack

import (
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func New(pCfg *providers.Config) (*Provider, error) {
	// check if the token was provided by the option. This way is deprecated since it does not pass the token as secret
	// FIXME: remove me in v8.0
	token := pCfg.Options["token"]

	// if a secret was provided, it always overrides the env variable since it has precedence
	if len(pCfg.Credentials) > 0 {
		for i := range pCfg.Credentials {
			cred := pCfg.Credentials[i]
			if cred.Type == vault.CredentialType_password {
				token = string(cred.Secret)
			} else {
				log.Warn().Str("credential-type", cred.Type.String()).Msg("unsupported credential type for Slack provider")
			}
		}
	}

	if token == "" {
		return nil, errors.New("a valid Slack token is required, pass --token '<yourtoken>' or set SLACK_TOKEN environment variable")
	}

	client := slack.New(token)

	teamInfo, err := client.GetTeamInfo()
	if err != nil {
		return nil, err
	}

	return &Provider{
		client:   client,
		opts:     pCfg.Options,
		teamInfo: teamInfo,
	}, nil
}

type Provider struct {
	client   *slack.Client
	opts     map[string]string
	teamInfo *slack.TeamInfo
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{}
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	return providers.RUNTIME_SLACK
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (p *Provider) Client() *slack.Client {
	return p.client
}
