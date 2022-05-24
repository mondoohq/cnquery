package gitlab

import (
	"errors"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/xanzy/go-gitlab"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
	"go.mondoo.io/mondoo/motor/vault"
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
		token = os.Getenv("GITLAB_TOKEN")
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
		return nil, errors.New("you need to provide GitLab token")
	}

	client, err := gitlab.NewClient(token)
	if err != nil {
		return nil, err
	}

	if tc.Options["group"] == "" {
		return nil, errors.New("you need to provide a group for gitlab")
	}

	return &Transport{
		client:    client,
		opts:      tc.Options,
		GroupPath: tc.Options["group"],
	}, nil
}

type Transport struct {
	client    *gitlab.Client
	opts      map[string]string
	GroupPath string
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("GitLab does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("GitLab does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_Gitlab,
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

func (t *Transport) Client() *gitlab.Client {
	return t.client
}
