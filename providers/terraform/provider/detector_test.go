// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectNameFromFile_Directory(t *testing.T) {
	name := parseNameFromPath("./testdata/nested")
	assert.Equal(t, "directory nested", name)
}

func TestDetectNameFromFile_File(t *testing.T) {
	name := parseNameFromPath("./testdata/nested/terraform.tfstate")
	assert.Equal(t, "terraform", name)
}

func TestDetectNameFromSsh(t *testing.T) {
	url := "git@gitlab.com:exampleorg/example-gitlab.git"
	domain, org, repo, err := parseSSHURL(url)
	require.NoError(t, err)
	assert.Equal(t, "gitlab.com", domain)
	assert.Equal(t, "exampleorg", org)
	assert.Equal(t, "example-gitlab", repo)
}

func TestDetectNameFromSsh_GitlabSubgroups(t *testing.T) {
	url := "git@gitlab.example.com:exampleorg/group/example-gitlab.git"
	domain, org, repo, err := parseSSHURL(url)
	require.NoError(t, err)
	assert.Equal(t, "gitlab.example.com", domain)
	assert.Equal(t, "exampleorg", org)
	assert.Equal(t, "example-gitlab", repo)
}
