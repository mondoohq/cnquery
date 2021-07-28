package github

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/google/go-github/v37/github"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
	"golang.org/x/oauth2"
)

func New(tc *transports.TransportConfig) (*Transport, error) {
	token := tc.Options["token"]
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
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
	return nil, errors.New("github does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("github does not implement FileInfo")
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

func (t *Transport) Client() *github.Client {
	return t.client
}
