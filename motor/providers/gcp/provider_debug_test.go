//go:build debugtest
// +build debugtest

package gcp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
)

func TestGcpDiscovery(t *testing.T) {
	orgId := "<insert org id>"
	pCfg := &providers.Config{
		Type: providers.ProviderType_GCP,
		Options: map[string]string{
			"organization": orgId,
		},
		Discover: &providers.Discovery{
			Targets: []string{"all"},
		},
	}

	p, err := New(pCfg)
	require.NoError(t, err)
	org, err := p.GetOrganization(orgId)
	require.NoError(t, err)

	projects, err := trans.GetProjectsForOrganization(org)
	require.NoError(t, err)
	fmt.Printf("%v", projects)
}
