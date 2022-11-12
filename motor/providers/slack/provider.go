package slack

import (
	"errors"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func New(tc *providers.Config) (*Provider, error) {
	// check if the token was provided by the option. This way is deprecated since it does not pass the token as secret
	token := tc.Options["token"]

	// if no token was provided, lets read the env variable
	if token == "" {
		token = os.Getenv("SLACK_TOKEN")
	}

	// if a secret was provided, it always overrides the env variable since it has precedence
	if len(tc.Credentials) > 0 {
		for i := range tc.Credentials {
			cred := tc.Credentials[i]
			if cred.Type == vault.CredentialType_password {
				token = string(cred.Secret)
			} else {
				log.Warn().Str("credential-type", cred.Type.String()).Msg("unsupported credential type for GitHub provider")
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
		opts:     tc.Options,
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
	return ""
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (p *Provider) Client() *slack.Client {
	return p.client
}
