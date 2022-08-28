package execruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGithubRuntimeEnv(t *testing.T) {
	// set mock provider
	environmentProvider = newMockEnvProvider()
	environmentProvider.Setenv("CI", "1")
	environmentProvider.Setenv("GITHUB_ACTION", "action")
	environmentProvider.Setenv("GITHUB_SHA", "1234")
	environmentProvider.Setenv("GITHUB_REF", "example-project")
	environmentProvider.Setenv("GITHUB_ACTOR", "johndoe")
	environmentProvider.Setenv("GITHUB_RUN_NUMBER", "23")

	env := Detect()
	assert.True(t, env.IsAutomatedEnv())
	assert.Equal(t, GITHUB, env.Id)
	assert.Equal(t, "GitHub Actions", env.Name)

	annotations := env.Labels()
	assert.Equal(t, 6, len(annotations))
	assert.Equal(t, "actions.github.com", annotations["mondoo.com/exec-environment"])
	assert.Equal(t, "action", annotations["actions.github.com/action"])
	assert.Equal(t, "1234", annotations["actions.github.com/sha"])
	assert.Equal(t, "example-project", annotations["actions.github.com/ref"])
	assert.Equal(t, "johndoe", annotations["actions.github.com/actor"])
	assert.Equal(t, "23", annotations["actions.github.com/run-number"])
}
