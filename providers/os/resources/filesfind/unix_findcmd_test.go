// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package filesfind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnixFilesCmdGeneration(t *testing.T) {
	depth1 := int64(1)
	depth10 := int64(10)
	depth0 := int64(0)

	tests := []struct {
		TestTitle   string
		From        string
		Xdev        bool
		FileType    string
		Regex       string
		Permission  int64
		Search      string
		Depth       *int64
		ExpectedCmd string
	}{
		{
			TestTitle:   "basic file search",
			From:        "/Users/john/.aws",
			FileType:    "file",
			ExpectedCmd: "find -L \"/Users/john/.aws\" -xdev -type f -perm -0",
		},
		{
			TestTitle:   "directory search with xdev enabled",
			From:        "/etc",
			Xdev:        true,
			FileType:    "directory",
			Permission:  0o777, // 0o777 means "no filter"
			ExpectedCmd: "find -L \"/etc\" -type d",
		},
		{
			TestTitle:   "search with depth (decimal, not octal)",
			From:        "/var/log",
			FileType:    "file",
			Permission:  0o777,
			Depth:       &depth10,
			ExpectedCmd: "find -L \"/var/log\" -xdev -type f -maxdepth 10",
		},
		{
			TestTitle:   "search with depth 0",
			From:        "/tmp",
			Permission:  0o777,
			Depth:       &depth0,
			ExpectedCmd: "find -L \"/tmp\" -xdev -maxdepth 0",
		},
		{
			TestTitle:   "search with depth 1",
			From:        "/home",
			FileType:    "file",
			Permission:  0o777,
			Depth:       &depth1,
			ExpectedCmd: "find -L \"/home\" -xdev -type f -maxdepth 1",
		},
		{
			TestTitle:   "search with name pattern (quoted)",
			From:        "/etc",
			Search:      "*.conf",
			Permission:  0o777,
			ExpectedCmd: "find -L \"/etc\" -xdev -name '*.conf'",
		},
		{
			TestTitle:   "search with regex",
			From:        "/var/log",
			Regex:       ".*\\.log$",
			Permission:  0o777,
			ExpectedCmd: "find -L \"/var/log\" -xdev -regex '.*\\.log$'",
		},
		{
			TestTitle:   "search with regex containing single quotes",
			From:        "/data",
			Regex:       ".*'test'.*",
			Permission:  0o777,
			ExpectedCmd: "find -L \"/data\" -xdev -regex '.*'\\''test'\\''.*'",
		},
		{
			TestTitle:   "search with name containing single quotes",
			From:        "/data",
			Search:      "file's name.txt",
			Permission:  0o777,
			ExpectedCmd: "find -L \"/data\" -xdev -name 'file'\\''s name.txt'",
		},
		{
			TestTitle:   "search with permissions",
			From:        "/home",
			Permission:  0o644,
			ExpectedCmd: "find -L \"/home\" -xdev -perm -644",
		},
		{
			TestTitle:   "search with all filters combined",
			From:        "/var",
			FileType:    "file",
			Regex:       ".*\\.log$",
			Permission:  0o644,
			Depth:       &depth1,
			ExpectedCmd: "find -L \"/var\" -xdev -type f -regex '.*\\.log$' -perm -644 -maxdepth 1",
		},
		{
			TestTitle:   "path with spaces",
			From:        "/Users/john/My Documents",
			FileType:    "file",
			Permission:  0o777,
			ExpectedCmd: "find -L \"/Users/john/My Documents\" -xdev -type f",
		},
		{
			TestTitle:   "symlink search",
			From:        "/usr/lib",
			FileType:    "link",
			Permission:  0o777,
			ExpectedCmd: "find -L \"/usr/lib\" -xdev -type l",
		},
		{
			TestTitle:   "socket search",
			From:        "/var/run",
			FileType:    "socket",
			Permission:  0o777,
			ExpectedCmd: "find -L \"/var/run\" -xdev -type s",
		},
		{
			TestTitle:   "block device search",
			From:        "/dev",
			FileType:    "block",
			Permission:  0o777,
			ExpectedCmd: "find -L \"/dev\" -xdev -type b",
		},
		{
			TestTitle:   "character device search",
			From:        "/dev",
			FileType:    "character",
			Permission:  0o777,
			ExpectedCmd: "find -L \"/dev\" -xdev -type c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.TestTitle, func(t *testing.T) {
			cmd := BuildFilesFindCmd(tt.From, tt.Xdev, tt.FileType, tt.Regex, tt.Permission, tt.Search, tt.Depth)
			assert.Equal(t, tt.ExpectedCmd, cmd)
		})
	}
}

func TestShellEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with'quote", "with'\\''quote"},
		{"'start", "'\\''start"},
		{"end'", "end'\\''"},
		{"multi'ple'quotes", "multi'\\''ple'\\''quotes"},
		{"no quotes here", "no quotes here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := shellEscape(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
