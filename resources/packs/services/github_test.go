//go:build debugtest
// +build debugtest

package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/github"
)

func githubTestQuery(t *testing.T, query string) []*llx.RawResult {
	trans, err := github.New(&providers.TransportConfig{
		Backend: providers.ProviderType_GITHUB,
		Options: map[string]string{
			"owner":      "mondoohq",
			"repository": "ranger-rpc",
			"token":      "<TOKEN HERE>",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	executor := initExecutionContext(m)
	return testQueryWithExecutor(t, executor, query, nil)
}

func TestResource_Github(t *testing.T) {
	t.Run("ms365 organization", func(t *testing.T) {
		res := githubTestQuery(t, "github.repository.branches.where( name == \"main\")[0].headCommit { * }")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, string(""), res[0].Data.Value)
	})
}
