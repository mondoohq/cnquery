package github

import (
	"time"

	"errors"
	"github.com/google/go-github/v49/github"
	"go.mondoo.com/cnquery/motor/providers"
	provider "go.mondoo.com/cnquery/motor/providers/github"
	"go.mondoo.com/cnquery/resources/packs/github/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func githubProvider(t providers.Instance) (*provider.Provider, error) {
	gt, ok := t.(*provider.Provider)
	if !ok {
		return nil, errors.New("github resource is not supported on this provider")
	}
	return gt, nil
}

func githubTimestamp(ts *github.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	return &ts.Time
}

const (
	paginationPerPage = 100
)
