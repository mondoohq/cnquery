// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccountIdsFromVolumePermissions(t *testing.T) {
	tests := []struct {
		name  string
		perms []any
		want  []any
	}{
		{
			name:  "shared with two accounts",
			perms: []any{map[string]any{"UserId": "111111111111"}, map[string]any{"UserId": "222222222222"}},
			want:  []any{"111111111111", "222222222222"},
		},
		{
			name:  "public group entry has no UserId",
			perms: []any{map[string]any{"Group": "all"}},
			want:  []any{},
		},
		{
			name:  "mixed public and account",
			perms: []any{map[string]any{"Group": "all"}, map[string]any{"UserId": "333333333333"}},
			want:  []any{"333333333333"},
		},
		{
			name:  "empty UserId skipped",
			perms: []any{map[string]any{"UserId": ""}},
			want:  []any{},
		},
		{
			name:  "no permissions",
			perms: []any{},
			want:  []any{},
		},
		{
			name:  "non-map entries ignored",
			perms: []any{"unexpected", map[string]any{"UserId": "444444444444"}},
			want:  []any{"444444444444"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, accountIdsFromVolumePermissions(tt.perms))
		})
	}
}
