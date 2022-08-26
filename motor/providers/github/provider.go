package github

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/google/go-github/v45/github"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/motor/providers"
	os_provider "go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/fsutil"
	"go.mondoo.com/cnquery/motor/vault"
	"golang.org/x/oauth2"
)

var (
	_ providers.Instance                  = (*Provider)(nil)
	_ providers.PlatformIdentifier        = (*Provider)(nil)
	_ os_provider.OperatingSystemProvider = (*Provider)(nil)
)

func New(tc *providers.Config) (*Provider, error) {
	// check if the token was provided by the option. This way is deprecated since it does not pass the token as secret
	token := tc.Options["token"]

	// if no token was provided, lets read the env variable
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
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
		return nil, errors.New("a valid GitHub token is required, pass --token '<yourtoken>' or set GITHUB_TOKEN environment variable")
	}

	var oauthClient *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		ctx := context.Background()
		oauthClient = oauth2.NewClient(ctx, ts)
	}

	client := github.NewClient(oauthClient)

	return &Provider{
		client: client,
		opts:   tc.Options,
	}, nil
}

type Provider struct {
	client *github.Client
	opts   map[string]string
}

func (p *Provider) RunCommand(command string) (*os_provider.Command, error) {
	return nil, providers.ErrRunCommandNotImplemented
}

func (p *Provider) FileInfo(path string) (os_provider.FileInfoDetails, error) {
	return os_provider.FileInfoDetails{}, providers.ErrFileInfoNotImplemented
}

func (p *Provider) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Github,
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

func (p *Provider) Client() *github.Client {
	return p.client
}
