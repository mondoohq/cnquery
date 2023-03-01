package terraform_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers"
	provider "go.mondoo.com/cnquery/motor/providers/terraform"
	"go.mondoo.com/cnquery/resources/packs/os"
	"go.mondoo.com/cnquery/resources/packs/terraform"
	"go.mondoo.com/cnquery/resources/packs/testutils"
)

var x = testutils.InitTester(testutils.LinuxMock(), os.Registry)

func testTerraformHclQuery(t *testing.T, path string, query string) []*llx.RawResult {
	p, err := provider.New(&providers.Config{
		Backend: providers.ProviderType_TERRAFORM,
		Options: map[string]string{
			"path": path,
		},
	})
	require.NoError(t, err)

	m, err := motor.New(p)
	require.NoError(t, err)

	x := testutils.InitTester(m, terraform.Registry)
	return x.TestQuery(t, query)
}

func testTerraformStateQuery(t *testing.T, query string) []*llx.RawResult {
	trans, err := provider.New(&providers.Config{
		Backend: providers.ProviderType_TERRAFORM,
		Options: map[string]string{
			"asset-type": "state",
			"path":       "./testdata/tfstate/state_aws_simple.json",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	x := testutils.InitTester(m, terraform.Registry)
	return x.TestQuery(t, query)
}

func testTerraformPlanQuery(t *testing.T, query string) []*llx.RawResult {
	trans, err := provider.New(&providers.Config{
		Backend: providers.ProviderType_TERRAFORM,
		Options: map[string]string{
			"asset-type": "plan",
			"path":       "./testdata/tfplan/plan_gcp_simple.json",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	x := testutils.InitTester(m, terraform.Registry)
	return x.TestQuery(t, query)
}
