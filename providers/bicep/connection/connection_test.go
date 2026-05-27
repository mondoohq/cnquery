// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func newTestConnection(t *testing.T, path string) (*BicepConnection, error) {
	t.Helper()
	asset := &inventory.Asset{
		Connections: []*inventory.Config{
			{
				Type:    "bicep",
				Options: map[string]string{"path": path},
			},
		},
	}
	return NewBicepConnection(0, asset, asset.Connections[0])
}

func TestDiscoverBicepParamFilesInDirectory(t *testing.T) {
	dir := filepath.Join("..", "resources", "testdata", "paramfiles")
	conn, err := newTestConnection(t, dir)
	require.NoError(t, err)

	// The directory holds both a .bicep and a .bicepparam file; each must land
	// in its own bucket. A .bicepparam ends in "param", not ".bicep", so the
	// .bicep walker never picks it up.
	require.Len(t, conn.BicepParamFiles(), 1)
	assert.True(t, filepath.Base(conn.BicepParamFiles()[0].Path) == "prod.bicepparam")
	assert.Contains(t, conn.BicepParamFiles()[0].Content, "using './main.bicep'")

	require.Len(t, conn.BicepFiles(), 1)
	assert.Equal(t, "main.bicep", filepath.Base(conn.BicepFiles()[0].Path))
}

func TestDiscoverSingleBicepParamFile(t *testing.T) {
	path := filepath.Join("..", "resources", "testdata", "paramfiles", "prod.bicepparam")
	conn, err := newTestConnection(t, path)
	require.NoError(t, err)

	require.Len(t, conn.BicepParamFiles(), 1)
	assert.Equal(t, path, conn.BicepParamFiles()[0].Path)
	assert.Empty(t, conn.BicepFiles())
}
