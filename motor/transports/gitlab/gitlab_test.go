// +build debugtest

package gitlab

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
)

func TestGitlab(t *testing.T) {
	trans, err := New(&transports.TransportConfig{
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
