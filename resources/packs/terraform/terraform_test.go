package terraform_test

import (
	"testing"

	"go.mondoo.io/mondoo/motor/providers/tfstate"
	"go.mondoo.io/mondoo/resources/packs/terraform"

	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/providers"
	provider "go.mondoo.io/mondoo/motor/providers/terraform"
	"go.mondoo.io/mondoo/resources/packs/os"
	"go.mondoo.io/mondoo/resources/packs/testutils"
)

var x = testutils.InitTester(testutils.LinuxMock(), os.Registry)

func testTerraformHclQuery(t *testing.T, query string) []*llx.RawResult {
	p, err := provider.New(&providers.Config{
		Backend: providers.ProviderType_TERRAFORM,
		Options: map[string]string{
			"path": "./testdata/terraform",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(p)
	require.NoError(t, err)

	x := testutils.InitTester(m, terraform.Registry)
	return x.TestQuery(t, query)
}

func testTerraformStateQuery(t *testing.T, query string) []*llx.RawResult {
	trans, err := tfstate.New(&providers.Config{
		Backend: providers.ProviderType_TERRAFORM_STATE,
		Options: map[string]string{
			"path": "./testdata/tfstate/state_simple.json",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	x := testutils.InitTester(m, terraform.Registry)
	return x.TestQuery(t, query)
}
