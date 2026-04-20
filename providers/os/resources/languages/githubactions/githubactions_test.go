// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package githubactions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUses(t *testing.T) {
	tests := []struct {
		input    string
		expected *ActionRef
	}{
		{"actions/checkout@v4", &ActionRef{Owner: "actions", Repo: "checkout", Ref: "v4"}},
		{"docker/build-push-action@v5.1.0", &ActionRef{Owner: "docker", Repo: "build-push-action", Ref: "v5.1.0"}},
		{"github/codeql-action/init@v3", &ActionRef{Owner: "github", Repo: "codeql-action", Path: "init", Ref: "v3"}},
		{"actions/checkout@abc123def456", &ActionRef{Owner: "actions", Repo: "checkout", Ref: "abc123def456"}},
		// Local actions — skip
		{"./my-action", nil},
		{"../shared/action", nil},
		// Docker actions — skip
		{"docker://alpine:3.18", nil},
		// Invalid
		{"actions/checkout", nil},
		{"justarepo@v1", nil},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseUses(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestActionRefName(t *testing.T) {
	assert.Equal(t, "actions/checkout", ActionRef{Owner: "actions", Repo: "checkout", Ref: "v4"}.Name())
	assert.Equal(t, "github/codeql-action/init", ActionRef{Owner: "github", Repo: "codeql-action", Path: "init", Ref: "v3"}.Name())
}

func TestNewPackageUrl(t *testing.T) {
	assert.Equal(t, "pkg:github/actions/checkout@v4", NewPackageUrl("actions", "checkout", "v4"))
	assert.Equal(t, "pkg:github/docker/build-push-action@v5.1.0", NewPackageUrl("docker", "build-push-action", "v5.1.0"))
}
