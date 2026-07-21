// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestExpandSshdGlob(t *testing.T) {
	fs := afero.NewMemMapFs()
	files := []string{
		"/etc/ssh/sshd_config",
		"/etc/ssh/decoy.conf",
		"/etc/ssh/sshd_config.d/10-a.conf",
		"/etc/ssh/sshd_config.d/20-b.conf",
		"/etc/ssh/sshd_config.d/nested/deep.conf",
	}
	for _, f := range files {
		require.NoError(t, afero.WriteFile(fs, f, []byte("Port 22\n"), 0o644))
	}
	afs := &afero.Afero{Fs: fs}

	tests := []struct {
		name string
		glob string
		want []string
	}{
		{
			name: "non-glob absolute path is returned as-is",
			glob: "/etc/ssh/sshd_config",
			want: []string{"/etc/ssh/sshd_config"},
		},
		{
			name: "non-glob relative path resolves from /etc/ssh",
			glob: "sshd_config",
			want: []string{"/etc/ssh/sshd_config"},
		},
		{
			name: "absolute glob in a subdirectory",
			glob: "/etc/ssh/sshd_config.d/*.conf",
			want: []string{"/etc/ssh/sshd_config.d/10-a.conf", "/etc/ssh/sshd_config.d/20-b.conf"},
		},
		{
			// Regression: a relative Include glob with a subdirectory must not
			// drop the subdirectory segment and glob one level too shallow.
			name: "relative glob in a subdirectory",
			glob: "sshd_config.d/*.conf",
			want: []string{"/etc/ssh/sshd_config.d/10-a.conf", "/etc/ssh/sshd_config.d/20-b.conf"},
		},
		{
			// Regression: a single-segment relative glob must expand within
			// /etc/ssh, not return the directory itself.
			name: "relative single-segment glob",
			glob: "*.conf",
			want: []string{"/etc/ssh/decoy.conf"},
		},
		{
			name: "glob does not descend into subdirectories",
			glob: "/etc/ssh/*.conf",
			want: []string{"/etc/ssh/decoy.conf"},
		},
		{
			name: "glob against a missing directory yields no matches",
			glob: "/etc/does-not-exist/*.conf",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandSshdGlob(afs, tt.glob)
			require.NoError(t, err)
			require.ElementsMatch(t, tt.want, got)
		})
	}
}
