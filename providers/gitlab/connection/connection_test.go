// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package connection

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

func TestGitlab(t *testing.T) {
	p, err := New(&inventory.Config{
		Options: map[string]string{
			"token": "<add-token-here>",
			"group": "mondoolabs",
		},
	})
	require.NoError(t, err)

	id, err := p.Identifier()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(id, "//platformid.api.mondoo.app/runtime/gitlab/group/"))
}
