package execruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitlabRuntimeEnv(t *testing.T) {
	// set mock provider
	environmentProvider = newMockEnvProvider()
	environmentProvider.Setenv("CI", "1")
	environmentProvider.Setenv("GITLAB_CI", "1")
	environmentProvider.Setenv("CI_PROJECT_URL", "https://example.com/project")
	environmentProvider.Setenv("CI_PROJECT_NAME", "example-project")
	environmentProvider.Setenv("CI_JOB_ID", "123456")
	environmentProvider.Setenv("GITLAB_USER_ID", "johndoe")

	env := Detect()
	assert.True(t, env.IsAutomatedEnv())
	assert.Equal(t, GITLAB, env.Id)
	assert.Equal(t, "GitLab CI", env.Name)

	annotations := env.Labels()
	assert.Equal(t, 5, len(annotations))
	assert.Equal(t, "gitlab.com", annotations["mondoo.com/exec-environment"])
	assert.Equal(t, "https://example.com/project", annotations["gitlab.com/project-url"])
	assert.Equal(t, "example-project", annotations["gitlab.com/project-name"])
	assert.Equal(t, "123456", annotations["gitlab.com/job-id"])
	assert.Equal(t, "johndoe", annotations["gitlab.com/user-id"])
}
