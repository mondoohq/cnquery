// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
)

func TestTerraform(t *testing.T) {
	p, err := NewHclConnection(0, &inventory.Asset{
		Connections: []*inventory.Config{
			{
				Options: map[string]string{
					"path": "./testdata/hcl",
				},
				Type: "hcl",
			},
		},
	})
	require.NoError(t, err)

	files := p.Parser().Files()
	assert.Equal(t, len(files), 2)
}

func TestModuleManifestIssue676(t *testing.T) {
	// See https://github.com/mondoohq/cnquery/issues/676
	p, err := NewHclConnection(0, &inventory.Asset{
		Connections: []*inventory.Config{
			{
				Options: map[string]string{
					"path": "./testdata/issue676",
				},
				Type: "hcl",
			},
		},
	})
	require.NoError(t, err)

	moduleManifest := p.ModulesManifest()
	require.NotNil(t, moduleManifest)
	require.Len(t, moduleManifest.Records, 3)
}
