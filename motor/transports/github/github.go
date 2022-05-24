package github

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/google/go-github/v43/github"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
	"go.mondoo.io/mondoo/motor/vault"
	"golang.org/x/oauth2"
)

var (
	_ transports.Transport                   = (*Transport)(nil)
	_ transports.TransportPlatformIdentifier = (*Transport)(nil)
)

func New(tc *transports.TransportConfig) (*Transport, error) {
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
				log.Warn().Str("credential-type", cred.Type.String()).Msg("unsupported credential type for GitHub transport")
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

	return &Transport{
		client: client,
		opts:   tc.Options,
	}, nil
}

type Transport struct {
	client *github.Client
	opts   map[string]string
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("GitHub does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("GitHub does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_Github,
	}
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return ""
}

func (t *Transport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{
		transports.TransportPlatformIdentifierDetector,
	}
}

func (t *Transport) Client() *github.Client {
	return t.client
}
