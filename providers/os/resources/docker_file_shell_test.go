// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseShellCommands(t *testing.T) {
	cases := []struct {
		name   string
		script string
		want   [][]string
	}{
		{
			name:   "single command",
			script: "apt-get install -y nginx",
			want:   [][]string{{"apt-get", "install", "-y", "nginx"}},
		},
		{
			name:   "and-chained commands",
			script: "apt-get update && apt-get install -y nginx",
			want: [][]string{
				{"apt-get", "update"},
				{"apt-get", "install", "-y", "nginx"},
			},
		},
		{
			name:   "pipe splits commands",
			script: "curl -fsSL https://example.com | sh",
			want: [][]string{
				{"curl", "-fsSL", "https://example.com"},
				{"sh"},
			},
		},
		{
			name:   "or and semicolon separators",
			script: "test -f x || touch x ; echo done",
			want: [][]string{
				{"test", "-f", "x"},
				{"touch", "x"},
				{"echo", "done"},
			},
		},
		{
			name:   "leading env assignment is stripped",
			script: "DEBIAN_FRONTEND=noninteractive apt-get install -y tzdata",
			want:   [][]string{{"apt-get", "install", "-y", "tzdata"}},
		},
		{
			name:   "multiple env assignments stripped",
			script: "A=1 B=2 make build",
			want:   [][]string{{"make", "build"}},
		},
		{
			name:   "operator inside double quotes is not a separator",
			script: `sh -c "a && b"`,
			want:   [][]string{{"sh", "-c", "a && b"}},
		},
		{
			name:   "single quotes keep content literal",
			script: `echo 'hello | world'`,
			want:   [][]string{{"echo", "hello | world"}},
		},
		{
			name:   "double quoted whitespace stays one token",
			script: `echo "hello world"`,
			want:   [][]string{{"echo", "hello world"}},
		},
		{
			name:   "line continuation joins the chain",
			script: "apt-get update && \\\n    apt-get install -y curl",
			want: [][]string{
				{"apt-get", "update"},
				{"apt-get", "install", "-y", "curl"},
			},
		},
		{
			name:   "newline separates commands",
			script: "echo a\necho b",
			want: [][]string{
				{"echo", "a"},
				{"echo", "b"},
			},
		},
		{
			name:   "backslash escapes a space into the word",
			script: `echo a\ b`,
			want:   [][]string{{"echo", "a b"}},
		},
		{
			name:   "empty script yields no commands",
			script: "   \n  ",
			want:   nil,
		},
		{
			name:   "concatenated quoted and bare segments form one token",
			script: `echo foo"bar"baz`,
			want:   [][]string{{"echo", "foobarbaz"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parseShellCommands(tc.script))
		})
	}
}

func TestStripEnvAssignments(t *testing.T) {
	assert.Equal(t, []string{"make"}, stripEnvAssignments([]string{"A=1", "make"}))
	assert.Equal(t, []string{"apt-get", "install"}, stripEnvAssignments([]string{"apt-get", "install"}))
	// A token that merely contains '=' but isn't a leading assignment is kept.
	assert.Equal(t, []string{"--opt=val"}, stripEnvAssignments([]string{"--opt=val"}))
	// Stops at the first non-assignment, keeping later '=' tokens.
	assert.Equal(t, []string{"env", "FOO=bar"}, stripEnvAssignments([]string{"env", "FOO=bar"}))
}
