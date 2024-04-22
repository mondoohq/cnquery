// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package urlx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitSshUrl(t *testing.T) {
	tests := []struct {
		url      string
		provider string
		org      string
		repo     string
	}{
		{"git@github.com:mondoohq/lunalectric.git", "github.com", "mondoohq", "lunalectric"},
		{"git@github.com:mondoohq/lunalectric", "github.com", "mondoohq", "lunalectric"},
	}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.url, func(t *testing.T) {
			a, b, c, err := ParseGitSshUrl(cur.url)
			require.NoError(t, err)
			assert.Equal(t, cur.provider, a)
			assert.Equal(t, cur.org, b)
			assert.Equal(t, cur.repo, c)
		})
	}
}
