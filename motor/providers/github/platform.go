package github

import (
	"context"
	"errors"

	"github.com/google/go-github/v43/github"
)

func (t *Provider) Identifier() (string, error) {
	// TODO: if no organization is provided, we need to see this from the user perspective
	orgId := t.opts["organization"]
	if orgId == "" {
		userId := t.opts["login"]
		return "//platformid.api.mondoo.app/runtime/github/user/" + userId, nil
	}
	return "//platformid.api.mondoo.app/runtime/github/organization/" + orgId, nil
}

func (t *Provider) Organization() (*github.Organization, error) {
	orgId := t.opts["organization"]
	if orgId != "" {
		org, _, err := t.Client().Organizations.Get(context.Background(), orgId)
		return org, err
	}
	return nil, errors.New("no organization provided")
}

func (t *Provider) User() (*github.User, error) {
	userId := t.opts["login"]
	if userId != "" {
		user, _, err := t.Client().Users.Get(context.Background(), userId)
		return user, err
	}
	return nil, errors.New("no user provided")
}
