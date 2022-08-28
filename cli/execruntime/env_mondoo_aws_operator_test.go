package execruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMondooOperatorRuntimeEnv(t *testing.T) {
	// TODO: add back tests to run Detect() and it correctly detecting "mondoo-aws-operator"
	// gl := environmentDef["mondoo-aws-operator"]
	// assert.NotNil(t, gl)
	gl := mondooAwsOperatorEnv
	assert.Equal(t, "mondoo-aws-operator", gl.Id)
	assert.Equal(t, "Mondoo AWS Operator", gl.Name)

	// set mondoo provider
	environmentProvider = newMockEnvProvider()
	environmentProvider.Setenv("AWS_LAMBDA_RUNTIME_API", "http://localhost:124")

	// TODO: use Detect() here and see if it's the "mondoo-aws-operator"
	assert.True(t, mondooAwsOperatorEnv.Detect())

	annotations := gl.Labels()
	assert.Equal(t, 1, len(annotations))
	assert.Equal(t, "aws-ops.mondoo.com", annotations["mondoo.com/exec-environment"])
}
