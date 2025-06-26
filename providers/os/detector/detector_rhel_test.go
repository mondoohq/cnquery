// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestGetActivatedRhelModules(t *testing.T) {
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
				"/etc/dnf/modules.d": "",
			},
			expected: []string{},
		},
		{
			name: "valid modules",
			files: map[string]string{
				"/etc/dnf/modules.d/maven.module": `[maven]
name=maven
stream=3.8
profiles=
state=enabled`,
			},
			expected: []string{"maven"},
		},
		{
			name: "disabled module",
			files: map[string]string{
				"/etc/dnf/modules.d/maven.module": `[maven]
name=maven
stream=3.8
profiles=
state=disabled`,
			},
			expected: []string{},
		},
		{
			name: "multiple modules",
			files: map[string]string{
				"/etc/dnf/modules.d/maven.module": `[maven]
name=maven
stream=3.8
profiles=
state=enabled`,
				"/etc/dnf/modules.d/python36.module": `[python36]
name=python36
stream=3.6
profiles=
state=enabled`,
			},
			expected: []string{"maven", "python36"},
		},
		{
			name: "invalid content",
			files: map[string]string{
				"/etc/dnf/modules.d/invalid.module": `invalid content`,
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
				err := fs.MkdirAll("/etc/dnf/modules.d", 0o755)
				assert.NoError(t, err)
			}

			// Create the files
			for path, content := range tt.files {
				if path == "/etc/dnf/modules.d" {
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
			result := getActivatedRhelModules(conn)

			// Compare results
			assert.Equal(t, tt.expected, result)
		})
	}
}
