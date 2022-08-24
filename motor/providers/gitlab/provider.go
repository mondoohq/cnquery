package gitlab

import (
	"errors"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
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
		token = os.Getenv("GITLAB_TOKEN")
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
		return nil, errors.New("you need to provide GitLab token")
	}

	client, err := gitlab.NewClient(token)
	if err != nil {
		return nil, err
	}

	if tc.Options["group"] == "" {
		return nil, errors.New("you need to provide a group for gitlab")
	}

	return &Provider{
		client:    client,
		opts:      tc.Options,
		GroupPath: tc.Options["group"],
	}, nil
}

type Provider struct {
	client    *gitlab.Client
	opts      map[string]string
	GroupPath string
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Gitlab,
	}
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

func (p *Provider) Client() *gitlab.Client {
	return p.client
}
