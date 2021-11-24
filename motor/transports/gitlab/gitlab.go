package gitlab

import (
	"errors"
	"os"

	"github.com/spf13/afero"
	"github.com/xanzy/go-gitlab"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

var _ transports.Transport = (*Transport)(nil)
var _ transports.TransportPlatformIdentifier = (*Transport)(nil)

func New(tc *transports.TransportConfig) (*Transport, error) {
	token := tc.Options["token"]
	if token == "" {
		token = os.Getenv("GITLAB_TOKEN")
	}

	if token == "" {
		return nil, errors.New("you need to provide gitlab token")
	}

	client, err := gitlab.NewClient(token)
	if err != nil {
		return nil, err
	}

	if tc.Options["group"] == "" {
		return nil, errors.New("you need to provide a group for gitlab transport")
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
	return nil, errors.New("gitlab does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("gitlab does not implement FileInfo")
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
