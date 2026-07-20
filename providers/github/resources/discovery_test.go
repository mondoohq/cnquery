// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/github/connection"
)

func TestReposFilter_Include(t *testing.T) {
	reposFilter := NewReposFilter(&inventory.Config{
		Options: map[string]string{
			connection.OPTION_REPOS: "repo1,repo2",
		},
	})
	assert.False(t, reposFilter.skipRepo("repo1"))
	assert.False(t, reposFilter.skipRepo("repo2"))
	assert.True(t, reposFilter.skipRepo("repo3"))
}

func TestReposFilter_Exclude(t *testing.T) {
	reposFilter := NewReposFilter(&inventory.Config{
		Options: map[string]string{
			connection.OPTION_REPOS_EXCLUDE: "repo1,repo2",
		},
	})
	assert.True(t, reposFilter.skipRepo("repo1"))
	assert.True(t, reposFilter.skipRepo("repo2"))
	assert.False(t, reposFilter.skipRepo("repo3"))
}

func TestReposFilter_Both(t *testing.T) {
	reposFilter := NewReposFilter(&inventory.Config{
		Options: map[string]string{
			connection.OPTION_REPOS:         "repo3,repo1",
			connection.OPTION_REPOS_EXCLUDE: "repo1,repo2",
		},
	})
	assert.False(t, reposFilter.skipRepo("repo1")) // include takes precedence
	assert.True(t, reposFilter.skipRepo("repo2"))
	assert.False(t, reposFilter.skipRepo("repo3"))
}

func TestHandleTargets(t *testing.T) {
	t.Run("all expands to repos, users, and the cheap per-repo IaC targets", func(t *testing.T) {
		// The clone-per-match IaC targets (cloudformation, dockerfiles, bicep,
		// helm, kustomize) are intentionally excluded from `all` and only run
		// on an explicit --discover <type>.
		got := handleTargets([]string{connection.DiscoveryAll})
		assert.Equal(t, []string{
			connection.DiscoveryRepos,
			connection.DiscoveryUsers,
			connection.DiscoveryTerraform,
			connection.DiscoveryK8sManifests,
		}, got)
	})

	t.Run("all wins even when mixed with explicit targets", func(t *testing.T) {
		got := handleTargets([]string{connection.DiscoveryRepos, connection.DiscoveryAll})
		assert.Equal(t, []string{
			connection.DiscoveryRepos,
			connection.DiscoveryUsers,
			connection.DiscoveryTerraform,
			connection.DiscoveryK8sManifests,
		}, got)
	})

	t.Run("opt-in IaC targets pass through unchanged", func(t *testing.T) {
		in := []string{
			connection.DiscoveryCloudformation,
			connection.DiscoveryDockerfiles,
			connection.DiscoveryBicep,
			connection.DiscoveryHelm,
			connection.DiscoveryKustomize,
		}
		assert.Equal(t, in, handleTargets(in))
	})

	t.Run("explicit targets pass through unchanged", func(t *testing.T) {
		in := []string{connection.DiscoveryRepos, connection.DiscoveryUsers}
		assert.Equal(t, in, handleTargets(in))
	})

	t.Run("empty stays empty", func(t *testing.T) {
		assert.Empty(t, handleTargets([]string{}))
	})
}
