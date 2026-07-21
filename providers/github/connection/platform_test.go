// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGithubOrgPlatform(t *testing.T) {
	pf := NewGithubOrgPlatform("mondoohq")
	assert.NotNil(t, pf)
	assert.Equal(t, "github-org", pf.Name)
	assert.Equal(t, "GitHub Organization", pf.Title)
	assert.Equal(t, "api", pf.Kind)
	assert.Equal(t, "github", pf.Runtime)
	assert.Equal(t, []string{"github"}, pf.Family)
	assert.Equal(t, []string{"saas", "github", "organization", "mondoohq", "organization"}, pf.TechnologyUrlSegments)
}

func TestNewGithubUserPlatform(t *testing.T) {
	pf := NewGithubUserPlatform("octocat")
	assert.NotNil(t, pf)
	assert.Equal(t, "github-user", pf.Name)
	assert.Equal(t, "api", pf.Kind)
	assert.Equal(t, []string{"saas", "github", "user"}, pf.TechnologyUrlSegments)
}

func TestNewGitHubRepoPlatform(t *testing.T) {
	pf := NewGitHubRepoPlatform("mondoohq", "cnquery")
	assert.NotNil(t, pf)
	assert.Equal(t, "github-repo", pf.Name)
	assert.Equal(t, "api", pf.Kind)
	assert.Equal(t, []string{"saas", "github", "organization", "mondoohq", "repository"}, pf.TechnologyUrlSegments)
}

func TestNewGithubOrgIdentifier(t *testing.T) {
	assert.Equal(t,
		"//platformid.api.mondoo.app/runtime/github/organization/mondoohq",
		NewGithubOrgIdentifier("mondoohq"),
	)
}

func TestNewGithubUserIdentifier(t *testing.T) {
	assert.Equal(t,
		"//platformid.api.mondoo.app/runtime/github/user/octocat",
		NewGithubUserIdentifier("octocat"),
	)
}

func TestNewGitHubRepoIdentifier(t *testing.T) {
	assert.Equal(t,
		"//platformid.api.mondoo.app/runtime/github/owner/mondoohq/repository/cnquery",
		NewGitHubRepoIdentifier("mondoohq", "cnquery"),
	)
}
