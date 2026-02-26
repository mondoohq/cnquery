// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package windows

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// mockLocalConnection implements shared.Connection and returns Type_Local
type mockLocalConnection struct{}

func (m *mockLocalConnection) ID() uint32                                         { return 0 }
func (m *mockLocalConnection) ParentID() uint32                                   { return 0 }
func (m *mockLocalConnection) RunCommand(command string) (*shared.Command, error) { return nil, nil }
func (m *mockLocalConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	return shared.FileInfoDetails{}, nil
}
func (m *mockLocalConnection) FileSystem() afero.Fs               { return afero.NewOsFs() }
func (m *mockLocalConnection) Name() string                       { return "mock-local" }
func (m *mockLocalConnection) Type() shared.ConnectionType        { return shared.Type_Local }
func (m *mockLocalConnection) Asset() *inventory.Asset            { return &inventory.Asset{} }
func (m *mockLocalConnection) UpdateAsset(asset *inventory.Asset) {}
func (m *mockLocalConnection) Capabilities() shared.Capabilities {
	return shared.Capability_None
}

func TestGetWindowsOSBuild_Integration(t *testing.T) {
	conn := &mockLocalConnection{}
	ver, err := GetWindowsOSBuild(conn)
	require.NoError(t, err)
	require.NotNil(t, ver)

	assert.NotEmpty(t, ver.CurrentBuild, "CurrentBuild should not be empty")
	assert.NotEmpty(t, ver.ProductName, "ProductName should not be empty")
	assert.NotEmpty(t, ver.Architecture, "Architecture should not be empty")
}
