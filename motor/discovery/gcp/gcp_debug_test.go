// +build debugtest

package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
	gcp_transport "go.mondoo.io/mondoo/motor/transports/gcp"
)

func TestGcpDiscovery(t *testing.T) {
	projectid, err := gcp_transport.GetCurrentProject()
	require.NoError(t, err)

	tc := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_GCP,
		Options: map[string]string{
			"project": projectid,
		},
		Discover: &transports.Discovery{
			Targets: []string{"all"},
		},
	}

	r := GcpProjectResolver{}
	assets, err := r.Resolve(tc)
	require.NoError(t, err)
	assert.True(t, len(assets) > 0)
}
