// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/github/connection"
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
