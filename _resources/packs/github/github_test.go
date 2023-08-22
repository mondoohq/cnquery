// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package github_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers"
	github_provider "go.mondoo.com/cnquery/motor/providers/github"
	"go.mondoo.com/cnquery/resources/packs/github"
	"go.mondoo.com/cnquery/resources/packs/testutils"
)

var x = testutils.InitTester(GithubProvider(), github.Registry)

func GithubProvider() *motor.Motor {
	p, err := github_provider.New(&providers.Config{
		Backend: providers.ProviderType_GITHUB,
		Options: map[string]string{
			"owner":      "mondoohq",
			"repository": "ranger-rpc",
			"token":      "<TOKEN HERE>",
		},
	})
	if err != nil {
		panic(err)
	}

	m, err := motor.New(p)
	if err != nil {
		panic(err)
	}
	return m
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
