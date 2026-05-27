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
	assert.Equal(t, []string{"saas", "github", "organization", "mondoohq", "organization"}, pf.TechnologyUrlSegments)
	// the helper must not mutate the package-level template
	assert.Nil(t, GithubOrgPlatform.TechnologyUrlSegments)
}

func TestNewGithubUserPlatform(t *testing.T) {
	pf := NewGithubUserPlatform("octocat")
	assert.NotNil(t, pf)
	assert.Equal(t, []string{"saas", "github", "user"}, pf.TechnologyUrlSegments)
	assert.Nil(t, GithubUserPlatform.TechnologyUrlSegments)
}

func TestNewGitHubRepoPlatform(t *testing.T) {
	pf := NewGitHubRepoPlatform("mondoohq", "cnquery")
	assert.NotNil(t, pf)
	assert.Equal(t, []string{"saas", "github", "organization", "mondoohq", "repository"}, pf.TechnologyUrlSegments)
	assert.Nil(t, GithubRepoPlatform.TechnologyUrlSegments)
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
