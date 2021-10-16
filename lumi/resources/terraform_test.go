package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/terraform"
)

func terraformTestQuery(t *testing.T, query string) []*llx.RawResult {
	trans, err := terraform.New(&transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_TERRAFORM,
		Options: map[string]string{
			"path": "./testdata/terraform",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	executor := initExecutor(m)
	return testQueryWithExecutor(t, executor, query, nil)
}

func TestResource_Terraform(t *testing.T) {
	t.Run("terraform providers", func(t *testing.T) {
		res := terraformTestQuery(t, "terraform.providers[0].type")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, string("provider"), res[0].Data.Value)
	})

	t.Run("terraform nested blocks", func(t *testing.T) {
		res := terraformTestQuery(t, "terraform.blocks.where( type == \"resource\" && labels.contains(\"aws_instance\"))[0].type")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, string("resource"), res[0].Data.Value)
	})
}
