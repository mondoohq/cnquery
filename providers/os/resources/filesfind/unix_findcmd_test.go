// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package filesfind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func ptrInt64(v int64) *int64 { return &v }

func TestUnixFilesCmdGeneration(t *testing.T) {
	tests := []struct {
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
			From:        "/Users/john/.aws",
			FileType:    "file",
			ExpectedCmd: "find -L \"/Users/john/.aws\" -xdev -type f -perm -0",
		},
		{
			// -maxdepth must be decimal: depth 12 stays 12, not octal 14.
			From:        "/etc",
			FileType:    "file",
			Depth:       ptrInt64(12),
			ExpectedCmd: "find -L \"/etc\" -xdev -type f -perm -0 -maxdepth 12",
		},
		{
			// -name is single-quoted so glob characters reach find instead of being expanded by the shell.
			From:        "/etc",
			FileType:    "file",
			Search:      "*.conf",
			ExpectedCmd: "find -L \"/etc\" -xdev -type f -perm -0 -name '*.conf'",
		},
		{
			// dotfile glob plus depth: the leading-dot pattern must reach find intact alongside -maxdepth.
			From:        "/home/user",
			FileType:    "file",
			Search:      ".*",
			Depth:       ptrInt64(1),
			ExpectedCmd: "find -L \"/home/user\" -xdev -type f -perm -0 -name '.*' -maxdepth 1",
		},
		{
			// single quotes prevent shell variable/command expansion of the name pattern.
			From:        "/etc",
			FileType:    "file",
			Search:      "$HOME*",
			ExpectedCmd: "find -L \"/etc\" -xdev -type f -perm -0 -name '$HOME*'",
		},
		{
			// an embedded single quote is escaped with the '\'' idiom.
			From:        "/etc",
			FileType:    "file",
			Search:      "a'b",
			ExpectedCmd: "find -L \"/etc\" -xdev -type f -perm -0 -name 'a'\\''b'",
		},
		{
			// regex is single-quoted as well (previously an unaddressed TODO).
			From:        "/etc",
			FileType:    "file",
			Regex:       ".*\\.conf$",
			ExpectedCmd: "find -L \"/etc\" -xdev -type f -regex '.*\\.conf$' -perm -0",
		},
	}

	for _, tt := range tests {
		cmd := BuildFilesFindCmd(tt.From, tt.Xdev, tt.FileType, tt.Regex, tt.Permission, tt.Search, tt.Depth)
		assert.Equal(t, tt.ExpectedCmd, cmd)
	}
}
