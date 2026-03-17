// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package processes

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSocketLink(t *testing.T) {
	tests := []struct {
		link      string
		wantInode int64
		wantMatch bool
	}{
		{"socket:[41866700]", 41866700, true},
		{"socket:[0]", 0, true},
		{"socket:[999999999]", 999999999, true},
		{"pipe:[12345]", 0, false},
		{"/dev/null", 0, false},
		{"", 0, false},
		{"socket:[]", 0, false},
		{"socket:[notanumber]", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.link, func(t *testing.T) {
			if !strings.HasPrefix(tt.link, "socket:[") || !strings.HasSuffix(tt.link, "]") {
				require.False(t, tt.wantMatch)
				return
			}
			inodeStr := tt.link[len("socket:[") : len(tt.link)-1]
			inode, err := strconv.ParseInt(inodeStr, 10, 64)
			if tt.wantMatch {
				require.NoError(t, err)
				require.Equal(t, tt.wantInode, inode)
			} else {
				require.Error(t, err)
			}
		})
	}
}
