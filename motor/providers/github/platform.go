package github

import (
	"context"

	"errors"
	"github.com/google/go-github/v49/github"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
)

var (
	GithubRepoPlatform = &platform.Platform{
		Name:    "github-repo",
		Title:   "GitHub Repository",
		Family:  []string{"github"},
		Kind:    providers.Kind_KIND_API,
		Runtime: providers.RUNTIME_GITHUB,
	}
	GithubUserPlatform = &platform.Platform{
		Name:    "github-user",
		Title:   "GitHub User",
		Family:  []string{"github"},
		Kind:    providers.Kind_KIND_API,
		Runtime: providers.RUNTIME_GITHUB,
	}
	GithubOrgPlatform = &platform.Platform{
		Name:    "github-org",
		Title:   "GitHub Organization",
		Family:  []string{"github"},
		Kind:    providers.Kind_KIND_API,
		Runtime: providers.RUNTIME_GITHUB,
	}
)

func (p *Provider) PlatformInfo() (*platform.Platform, error) {
	if orgId := p.opts["organization"]; orgId != "" {
		return GithubOrgPlatform, nil
	}

	if userId := p.opts["user"]; userId != "" {
		return GithubUserPlatform, nil
	}

	_, err := p.Repository()
	if err == nil {
		return GithubRepoPlatform, nil
	}

	return nil, errors.Join(err, errors.New("could not detect GitHub asset type"))
}

func NewGithubOrgIdentifier(orgId string) string {
	return "//platformid.api.mondoo.app/runtime/github/organization/" + orgId
}

func NewGithubUserIdentifier(userId string) string {
	return "//platformid.api.mondoo.app/runtime/github/user/" + userId
}

func NewGitHubRepoIdentifier(ownerId string, repoId string) string {
	return "//platformid.api.mondoo.app/runtime/github/owner/" + ownerId + "/repository/" + repoId
}

func (p *Provider) Identifier() (string, error) {
	orgId := p.opts["organization"]
	if orgId != "" {
		return NewGithubOrgIdentifier(orgId), nil
	}

	userId := p.opts["user"]
	if userId != "" {
		return NewGithubUserIdentifier(userId), nil
	}

	repoId := p.opts["repository"]
	if repoId != "" {
		ownerId := p.opts["owner"]
		if ownerId == "" {
			ownerId = p.opts["organization"]
		}
		if ownerId == "" {
			ownerId = p.opts["user"]
		}
		return NewGitHubRepoIdentifier(ownerId, repoId), nil
	}

	return "", errors.New("could not identifier GitHub asset")
}

func (p *Provider) Organization() (*github.Organization, error) {
	orgId := p.opts["organization"]
	if orgId == "" {
		orgId = p.opts["owner"]
	}
	if orgId != "" {
		org, _, err := p.Client().Organizations.Get(context.Background(), orgId)
		return org, err
	}

	return nil, errors.New("no organization provided")
}

func (p *Provider) User() (*github.User, error) {
	userId := p.opts["user"]
	if userId == "" {
		userId = p.opts["owner"]
	}

	if userId != "" {
		user, _, err := p.Client().Users.Get(context.Background(), userId)
		return user, err
	}
	return nil, errors.New("no user provided")
}

func (p *Provider) Repository() (*github.Repository, error) {
	ownerId := p.opts["owner"]
	if ownerId == "" {
		ownerId = p.opts["organization"]
	}
	if ownerId == "" {
		ownerId = p.opts["user"]
	}

	repoId := p.opts["repository"]
	if ownerId != "" && repoId != "" {
		repo, _, err := p.Client().Repositories.Get(context.Background(), ownerId, repoId)
		return repo, err
	}
	return nil, errors.New("no user provided")
}
