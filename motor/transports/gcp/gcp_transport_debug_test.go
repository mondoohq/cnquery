// +build debugtest

package gcp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
)

func TestGcpDiscovery(t *testing.T) {
	orgId := "<insert org id>"
	tc := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_GCP,
		Options: map[string]string{
			"organization": orgId,
		},
		Discover: &transports.Discovery{
			Targets: []string{"all"},
		},
	}

	trans, err := New(tc)
	require.NoError(t, err)
	org, err := trans.GetOrganization(orgId)
	require.NoError(t, err)

	projects, err := trans.GetProjectsForOrganization(org)
	require.NoError(t, err)
	fmt.Printf("%v", projects)
}
