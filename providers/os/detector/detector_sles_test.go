// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

type mockConnection struct {
	fs afero.Fs
}

func (m *mockConnection) ID() uint32 {
	return 1
}

func (m *mockConnection) ParentID() uint32 {
	return 0
}

func (m *mockConnection) FileSystem() afero.Fs {
	return m.fs
}

func (m *mockConnection) RunCommand(command string) (*shared.Command, error) {
	return nil, nil
}

func (m *mockConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	return shared.FileInfoDetails{}, nil
}

func (m *mockConnection) Name() string {
	return "mock"
}

func (m *mockConnection) Type() shared.ConnectionType {
	return shared.Type_Local
}

func (m *mockConnection) Asset() *inventory.Asset {
	return &inventory.Asset{}
}

func (m *mockConnection) UpdateAsset(asset *inventory.Asset) {}

func (m *mockConnection) Capabilities() shared.Capabilities {
	return shared.Capability_File
}

func TestGetActivatedSlesModules(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected []string
	}{
		{
			name:     "no modules directory",
			files:    map[string]string{},
			expected: []string{},
		},
		{
			name: "empty modules directory",
			files: map[string]string{
				"/etc/products.d": "",
			},
			expected: []string{},
		},
		{
			name: "valid modules",
			files: map[string]string{
				"/etc/products.d/base.prod": `<?xml version="1.0" encoding="UTF-8"?>
<product>
  <summary>SUSE Linux Enterprise Server</summary>
  <register>
    <target>sles</target>
  </register>
</product>`,
				"/etc/products.d/desktop.prod": `<?xml version="1.0" encoding="UTF-8"?>
<product>
  <summary>Basesystem Module</summary>
  <register>
    <target>sles</target>
    <flavor>module</flavor>
  </register>
</product>`,
				"/etc/products.d/we.prod": `<?xml version="1.0" encoding="UTF-8"?>
<product>
  <summary>SUSE Linux Enterprise High Availability Extension 15 SP5</summary>
  <register>
    <target>sles</target>
    <flavor>extension</flavor>
  </register>
</product>`,
			},
			expected: []string{"Basesystem", "Linux Enterprise High Availability Extension 15 SP5"},
		},
		{
			name: "invalid xml",
			files: map[string]string{
				"/etc/products.d/invalid.prod": `invalid xml content`,
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock filesystem
			fs := afero.NewMemMapFs()

			// Create the directory structure
			if len(tt.files) > 0 {
				err := fs.MkdirAll("/etc/products.d", 0o755)
				assert.NoError(t, err)
			}

			// Create the files
			for path, content := range tt.files {
				if path == "/etc/products.d" {
					continue
				}
				err := afero.WriteFile(fs, path, []byte(content), 0o644)
				assert.NoError(t, err)
			}

			// Create a mock connection
			conn := &mockConnection{
				fs: fs,
			}

			// Call the function
			result := getActivatedSlesModules(conn)

			// Compare results
			assert.Equal(t, tt.expected, result)
		})
	}
}
