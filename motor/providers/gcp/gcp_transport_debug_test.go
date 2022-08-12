//go:build debugtest
// +build debugtest

package gcp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers"
)

func TestGcpDiscovery(t *testing.T) {
	orgId := "<insert org id>"
	tc := &providers.TransportConfig{
		Backend: providers.ProviderType_GCP,
		Options: map[string]string{
			"organization": orgId,
		},
		Discover: &providers.Discovery{
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
