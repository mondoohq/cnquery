// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectNameFromFile(t *testing.T) {
	name := parseNameFromPath("/test/path/nested/terraform.tfstate")
	assert.Equal(t, "nested", name)
}

func TestDetectNameFromSsh(t *testing.T) {
	url := "git@gitlab.com:exampleorg/example-gitlab.git"
	domain, org, repo, err := parseSSHURL(url)
	require.NoError(t, err)
	assert.Equal(t, "gitlab.com", domain)
	assert.Equal(t, "exampleorg", org)
	assert.Equal(t, "example-gitlab", repo)
}
