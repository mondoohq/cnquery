package execruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAzureRuntimeEnv(t *testing.T) {
	// set mock provider
	environmentProvider = newMockEnvProvider()
	environmentProvider.Setenv("CI", "1")
	environmentProvider.Setenv("TF_BUILD", "1")
	environmentProvider.Setenv("BUILD_REPOSITORY_NAME", "example-project")
	environmentProvider.Setenv("BUILD_BUILDID", "1")
	environmentProvider.Setenv("BUILD_SOURCEVERSION", "897248974893749873894789374")
	environmentProvider.Setenv("BUILD_SOURCEVERSIONAUTHOR", "vj")

	env := Detect()
	assert.True(t, env.IsAutomatedEnv())
	assert.Equal(t, AZUREPIPELINE, env.Id)
	assert.Equal(t, "Azure Pipelines", env.Name)

	annotations := env.Labels()
	assert.Equal(t, 5, len(annotations))
	assert.Equal(t, "devops.azure.com", annotations["mondoo.com/exec-environment"])
	assert.Equal(t, "example-project", annotations["devops.azure.com/repository-name"])
	assert.Equal(t, "1", annotations["devops.azure.com/buildid"])
	assert.Equal(t, "vj", annotations["devops.azure.com/sourceversionauthor"])
}
