package github

import (
	"context"
	"errors"

	"github.com/google/go-github/v43/github"
)

func (p *Provider) Identifier() (string, error) {
	// TODO: if no organization is provided, we need to see this from the user perspective
	orgId := p.opts["organization"]
	if orgId != "" {
		return "//platformid.api.mondoo.app/runtime/github/organization/" + orgId, nil
	}

	repoId := p.opts["repository"]
	if repoId != "" {
		return "//platformid.api.mondoo.app/runtime/github/repository/" + repoId, nil
	}

	userId := p.opts["login"]
	return "//platformid.api.mondoo.app/runtime/github/user/" + userId, nil
}

func (p *Provider) Organization() (*github.Organization, error) {
	orgId := p.opts["organization"]
	if orgId != "" {
		org, _, err := p.Client().Organizations.Get(context.Background(), orgId)
		return org, err
	}
	return nil, errors.New("no organization provided")
}

func (p *Provider) Repository() (*github.Repository, error) {
	orgId := p.opts["organization"]
	repoId := p.opts["repository"]
	if orgId != "" && repoId != "" {
		repo, _, err := p.Client().Repositories.Get(context.Background(), orgId, repoId)
		return repo, err
	}
	return nil, errors.New("no user provided")
}

func (p *Provider) User() (*github.User, error) {
	userId := p.opts["login"]
	if userId != "" {
		user, _, err := p.Client().Users.Get(context.Background(), userId)
		return user, err
	}
	return nil, errors.New("no user provided")
}
