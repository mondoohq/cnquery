package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
)

func TestGithub(t *testing.T) {
	trans, err := New(&transports.TransportConfig{})
	require.NoError(t, err)

	client := trans.Client()

	ctx := context.Background()
	org, _, err := client.Organizations.Get(ctx, "mondoolabs")
	require.NoError(t, err)
	require.NotNil(t, org)
}
