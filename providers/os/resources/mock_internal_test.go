// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// mockConn implements shared.Connection with only the Asset() method populated.
type mockConn struct {
	asset *inventory.Asset
}

func (m *mockConn) ID() uint32                                         { return 0 }
func (m *mockConn) ParentID() uint32                                   { return 0 }
func (m *mockConn) RunCommand(command string) (*shared.Command, error) { return nil, nil }
func (m *mockConn) FileInfo(path string) (shared.FileInfoDetails, error) {
	return shared.FileInfoDetails{}, nil
}
func (m *mockConn) FileSystem() afero.Fs               { return nil }
func (m *mockConn) Name() string                       { return "mock" }
func (m *mockConn) Type() shared.ConnectionType        { return "mock" }
func (m *mockConn) Asset() *inventory.Asset            { return m.asset }
func (m *mockConn) UpdateAsset(asset *inventory.Asset) {}
func (m *mockConn) Capabilities() shared.Capabilities  { return 0 }

// connWithPlatform returns a mockConn with the given platform name set.
func connWithPlatform(name string) *mockConn {
	return &mockConn{
		asset: &inventory.Asset{
			Platform: &inventory.Platform{
				Name: name,
			},
		},
	}
}
