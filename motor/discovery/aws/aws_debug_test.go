//go:build debugtest
// +build debugtest

package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	aws_transport "go.mondoo.io/mondoo/motor/providers/aws"
)

func TestEC2Discovery(t *testing.T) {
	tc := &providers.TransportConfig{
		Backend: providers.ProviderType_AWS,
		Options: map[string]string{
			"profile": "mondoo-demo",
			"region":  "us-east-1",
		},
	}

	trans, err := aws_transport.New(tc, aws_transport.TransportOptions(tc.Options)...)
	require.NoError(t, err)

	r, err := NewEc2Discovery(trans.Config())
	require.NoError(t, err)

	assets, err := r.List()
	require.NoError(t, err)
	assert.True(t, len(assets) > 0)
}
