//go:build debugtest
// +build debugtest

package gitlab

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
)

func TestGitlabGroup(t *testing.T) {
	token := os.Getenv("GITLAB_TOKEN")
	p, err := New(&providers.Config{
		Options: map[string]string{
			"token": token,
			"group": "my-group",
		},
	})
	require.NoError(t, err)

	id, err := p.Identifier()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(id, "//platformid.api.mondoo.app/runtime/gitlab/group/"))
}

func TestGitlabProject(t *testing.T) {
	token := os.Getenv("GITLAB_TOKEN")
	p, err := New(&providers.Config{
		Options: map[string]string{
			"token":   token,
			"group":   "my-group",
			"project": "my-repo",
		},
	})
	require.NoError(t, err)

	id, err := p.Identifier()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(id, "//platformid.api.mondoo.app/runtime/gitlab/group/"))
}
