// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePacmanOwner(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"owned", "/usr/bin/claude is owned by claude-code 2.1.191-1\n", "claude-code"},
		{"name with plus", "/usr/bin/gcc is owned by gcc-libs 14.1.1-1\n", "gcc-libs"},
		{"empty", "", ""},
		{"unrelated", "error: No package owns /tmp/foo\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parsePacmanOwner(tt.output))
		})
	}
}

func TestParseDpkgOwner(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"simple", "bash: /bin/bash\n", "bash"},
		{"multiarch", "libc6:amd64: /lib/x86_64-linux-gnu/libc.so.6\n", "libc6"},
		{"skip diversion", "diversion by glx-diversions from: /usr/lib/libGL.so.1\nlibgl1:amd64: /usr/lib/libGL.so.1\n", "libgl1"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseDpkgOwner(tt.output))
		})
	}
}

func TestParseRpmOwner(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"owned", "bash\n", "bash"},
		{"hyphenated name", "python3-pip\n", "python3-pip"},
		{"not owned notice", "file /tmp/foo is not owned by any package\n", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseRpmOwner(tt.output))
		})
	}
}

func TestParseApkOwner(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"owned", "/usr/bin/claude is owned by claude-code-2.1.191-r0\n", "claude-code"},
		{"simple name", "/bin/busybox is owned by busybox-1.36.1-r15\n", "busybox"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseApkOwner(tt.output))
		})
	}
}

func TestApkStripVersion(t *testing.T) {
	assert.Equal(t, "claude-code", apkStripVersion("claude-code-2.1.191-r0"))
	assert.Equal(t, "busybox", apkStripVersion("busybox-1.36.1-r15"))
	assert.Equal(t, "py3-setuptools", apkStripVersion("py3-setuptools-68.0.0-r0"))
	// Package name ending in a digit-segment must not be confused with the
	// version boundary.
	assert.Equal(t, "gtk4", apkStripVersion("gtk4-4.14.0-r0"))
	// apk uses "_" (not "-") for pre-release suffixes, so the real name is
	// recovered even when the version carries an "_rc1"; the "-r" there must
	// not be taken as the release separator.
	assert.Equal(t, "foo", apkStripVersion("foo-1.0_rc1-r0"))
	// A package name ending in "-r<word>" (not digits) with no release suffix
	// must not be truncated at that "-r".
	assert.Equal(t, "super-r2d2", apkStripVersion("super-r2d2-1.0"))
}

func TestFirstLine(t *testing.T) {
	assert.Equal(t, "one", firstLine("one\ntwo\n"))
	assert.Equal(t, "solo", firstLine("solo"))
	assert.Equal(t, "", firstLine(""))
}

func TestShellQuote(t *testing.T) {
	assert.Equal(t, "'/usr/bin/claude'", shellQuote("/usr/bin/claude"))
	assert.Equal(t, `'/tmp/a'\''b'`, shellQuote("/tmp/a'b"))
}

func TestSafeBinaryName(t *testing.T) {
	// real tool binaries
	for _, ok := range []string{"claude", "codex", "qwen", "gemini-cli", "py3.11", "g++"} {
		assert.True(t, safeBinaryName.MatchString(ok), ok)
	}
	// anything with a slash, whitespace, or shell metacharacter is rejected
	for _, bad := range []string{"", "a/b", "a b", "$(rm -rf /)", "a;b", "a`b`", "a|b", "a&b", "a\nb"} {
		assert.False(t, safeBinaryName.MatchString(bad), bad)
	}
}
