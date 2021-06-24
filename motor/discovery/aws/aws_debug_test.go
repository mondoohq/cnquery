// +build debugtest

package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

func TestEC2Discovery(t *testing.T) {
	tc := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_AWS,
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
