// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/testutils"
	"go.mondoo.com/cnquery/v9/providers/github"
)

var x = testutils.InitTester(GithubProvider(), github.Registry)

func GithubProvider() *github.GithubConnection {
	p, err := github.NewGithubConnection(&inventory.Config{
		Backend: "github",
		Options: map[string]string{
			"owner":      "mondoohq",
			"repository": "ranger-rpc",
			"token":      "<TOKEN HERE>",
		},
	})
	if err != nil {
		panic(err)
	}

	return p.Connection
}

func TestResource_GithubRepo(t *testing.T) {
	t.Run("github project", func(t *testing.T) {
		res := x.TestQuery(t, "github.repository")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, string(""), res[0].Data.Value)
	})
}

func TestResource_Github(t *testing.T) {
	t.Run("github branch", func(t *testing.T) {
		res := x.TestQuery(t, "github.repository.branches.where( name == \"main\")[0].headCommit { * }")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, string(""), res[0].Data.Value)
	})
}
