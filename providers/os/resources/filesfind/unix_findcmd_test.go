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
	}

	for _, tt := range tests {
		cmd := BuildFilesFindCmd(tt.From, tt.Xdev, tt.FileType, tt.Regex, tt.Permission, tt.Search, tt.Depth)
		assert.Equal(t, tt.ExpectedCmd, cmd)
	}
}
