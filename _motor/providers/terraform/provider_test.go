// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
)

func TestTerraform(t *testing.T) {
	p, err := New(&providers.Config{
		Options: map[string]string{
			"path": "./testdata/hcl",
		},
	})
	require.NoError(t, err)

	files := p.Parser().Files()
	assert.Equal(t, len(files), 2)
}

func TestModuleManifestIssue676(t *testing.T) {
	// See https://github.com/mondoohq/cnquery/issues/676
	p, err := New(&providers.Config{
		Options: map[string]string{
			"path": "./testdata/issue676",
		},
	})
	require.NoError(t, err)

	require.NotNil(t, p.modulesManifest)
	require.Len(t, p.modulesManifest.Records, 3)
}
