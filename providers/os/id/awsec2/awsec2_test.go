// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package awsec2

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers/os/connection/fs"
	"go.mondoo.com/mql/providers/os/connection/shared"
)

// newFsConnection builds a filesystem connection over an empty in-memory
// filesystem, carrying the given connection options.
func newFsConnection(t *testing.T, options map[string]string) shared.Connection {
	t.Helper()

	conf := &inventory.Config{
		Type:    shared.Type_FileSystem.String(),
		Host:    "/mnt/host",
		Options: options,
	}
	asset := &inventory.Asset{Connections: []*inventory.Config{conf}}

	conn, err := fs.NewFileSystemConnectionWithFs(0, conf, asset, "/mnt/host", nil, afero.NewMemMapFs())
	require.NoError(t, err)
	return conn
}

func TestIsHostRootMount(t *testing.T) {
	t.Run("opted in", func(t *testing.T) {
		conn := newFsConnection(t, map[string]string{shared.HostRootOption: "true"})
		assert.True(t, isHostRootMount(conn))
	})

	t.Run("no options at all", func(t *testing.T) {
		// A plain `cnspec scan filesystem --path ...`: the mount may be a
		// snapshot of an entirely different machine, so IMDS must not be
		// consulted.
		conn := newFsConnection(t, nil)
		assert.False(t, isHostRootMount(conn))
	})

	t.Run("option present but not true", func(t *testing.T) {
		conn := newFsConnection(t, map[string]string{shared.HostRootOption: "false"})
		assert.False(t, isHostRootMount(conn))
	})

	t.Run("asset without connections", func(t *testing.T) {
		conf := &inventory.Config{
			Type:    shared.Type_FileSystem.String(),
			Host:    "/mnt/host",
			Options: map[string]string{shared.HostRootOption: "true"},
		}
		conn, err := fs.NewFileSystemConnectionWithFs(
			0, conf, &inventory.Asset{}, "/mnt/host", nil, afero.NewMemMapFs())
		require.NoError(t, err)

		assert.False(t, isHostRootMount(conn))
	})
}
