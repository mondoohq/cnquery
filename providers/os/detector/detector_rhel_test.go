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

func TestGetActivatedRhelSupportLevels(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected []string
	}{
		{
			name:     "no repos directory",
			files:    map[string]string{},
			expected: []string{},
		},
		{
			name: "empty repos directory",
			files: map[string]string{
				"/etc/yum.repos.d": "",
			},
			expected: []string{},
		},
		{
			name: "eus repos",
			files: map[string]string{
				"/etc/yum.repos.d/rhel.repo": `[rhel-8-for-x86_64-baseos-eus-rpms]
name=Red Hat Enterprise Linux 8 for x86_64 - BaseOS - Extended Update Support (RPMs)
enabled=1`,
			},
			expected: []string{"eus"},
		},
		{
			name: "disabled eus repo",
			files: map[string]string{
				"/etc/yum.repos.d/rhel.repo": `[rhel-8-for-x86_64-baseos-eus-rpms]
name=Red Hat Enterprise Linux 8 for x86_64 - BaseOS - Extended Update Support (RPMs)
enabled=0`,
			},
			expected: []string{},
		},
		{
			name: "multiple repos",
			files: map[string]string{
				"/etc/yum.repos.d/1.repo": `[rhui-rhel-8-for-x86_64-baseos-e4s-rhui-rpms]
name=Red Hat Enterprise Linux 8 for x86_64 - BaseOS - Update Services for SAP Solutions from RHUI (RPMs)
enabled=1
[rhui-rhel-8-for-x86_64-appstream-e4s-rhui-rpms]
name=Red Hat Enterprise Linux 8 for x86_64 - AppStream - Update Services for SAP Solutions from RHUI (RPMs)
enabled=1
`,
				"/etc/yum.repos.d/2.repo": `[rhel-8-for-x86_64-baseos-eus-rpms]
name=Red Hat Enterprise Linux 8 for x86_64 - BaseOS - Extended Update Support (RPMs)
enabled=1`,
			},
			expected: []string{"e4s", "eus"},
		},
		{
			name: "invalid content",
			files: map[string]string{
				"/etc/yum.repos.d/rhel.repo": `invalid content`,
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
				err := fs.MkdirAll("/etc/yum.repos.d", 0o755)
				assert.NoError(t, err)
			}

			// Create the files
			for path, content := range tt.files {
				if path == "/etc/yum.repos.d" {
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
			result := getActivatedRhelSupportLevels(conn)

			// Compare results
			assert.Equal(t, tt.expected, result)
		})
	}
}
