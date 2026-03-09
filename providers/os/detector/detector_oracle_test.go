// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestHasOracleELSEnabled(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected bool
	}{
		{
			name:     "no repos directory",
			files:    map[string]string{},
			expected: false,
		},
		{
			name: "empty repos directory",
			files: map[string]string{
				"/etc/yum.repos.d": "",
			},
			expected: false,
		},
		{
			name: "els repos enabled",
			files: map[string]string{
				"/etc/yum.repos.d/oracle-linux-ol7.repo": `[ol7_latest]
name=Oracle Linux 7Server Latest (x86_64)
enabled=1

[ol7_latest_ELS]
name=Oracle Linux 7Server ELS (x86_64)
enabled=1

[ol7_UEKR6]
name=Latest Unbreakable Enterprise Kernel Release 6 for Oracle Linux 7Server (x86_64)
enabled=1

[ol7_UEKR6_ELS]
name=Unbreakable Enterprise Kernel Release 6 for Oracle Linux 7 ELS (x86_64)
enabled=1
`,
			},
			expected: true,
		},
		{
			name: "els repos disabled",
			files: map[string]string{
				"/etc/yum.repos.d/oracle-linux-ol7.repo": `[ol7_latest_ELS]
name=Oracle Linux 7Server ELS (x86_64)
enabled=0

[ol7_UEKR6_ELS]
name=Unbreakable Enterprise Kernel Release 6 for Oracle Linux 7 ELS (x86_64)
enabled=0
`,
			},
			expected: false,
		},
		{
			name: "no els repos",
			files: map[string]string{
				"/etc/yum.repos.d/oracle-linux-ol7.repo": `[ol7_latest]
name=Oracle Linux 7Server Latest (x86_64)
enabled=1

[ol7_UEKR6]
name=Latest Unbreakable Enterprise Kernel Release 6 for Oracle Linux 7Server (x86_64)
enabled=1
`,
			},
			expected: false,
		},
		{
			name: "ksplice els repo",
			files: map[string]string{
				"/etc/yum.repos.d/oracle-linux-ol7.repo": `[ol7_ksplice_ELS]
name=Ksplice for Oracle Linux 7 ELS (x86_64)
enabled=1
`,
			},
			expected: true,
		},
		{
			name: "invalid content",
			files: map[string]string{
				"/etc/yum.repos.d/oracle-linux-ol7.repo": `invalid content`,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			if len(tt.files) > 0 {
				err := fs.MkdirAll("/etc/yum.repos.d", 0o755)
				assert.NoError(t, err)
			}

			for path, content := range tt.files {
				if path == "/etc/yum.repos.d" {
					continue
				}
				err := afero.WriteFile(fs, path, []byte(content), 0o644)
				assert.NoError(t, err)
			}

			conn := &mockConnection{
				fs: fs,
			}

			result := hasOracleELSEnabled(conn)

			assert.Equal(t, tt.expected, result)
		})
	}
}
