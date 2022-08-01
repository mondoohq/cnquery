//go:build debugtest
// +build debugtest

package gitlab

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers"
)

func TestGitlab(t *testing.T) {
	trans, err := New(&providers.TransportConfig{
		Options: map[string]string{
			"token": "<add-token-here>",
			"group": "mondoolabs",
		},
	})
	require.NoError(t, err)

	id, err := trans.Identifier()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(id, "//platformid.api.mondoo.app/runtime/gitlab/group/"))
}
