package execruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectGitlab(t *testing.T) {
	gl := environmentDef["gitlab"]
	assert.NotNil(t, gl)
	assert.Equal(t, "gitlab", gl.Id)
	assert.Equal(t, "GitLab CI", gl.Name)

	assert.False(t, gl.Detect())

	// set mock provider
	environmentProvider = newMockEnvProvider()
	environmentProvider.Setenv("CI", "1")
	environmentProvider.Setenv("GITLAB_CI", "1")
	assert.True(t, gl.Detect())

	environmentProvider.Setenv("CI_JOB_NAME", "test-job")
	environmentProvider.Setenv("GITLAB_USER_ID", "testuser")
	annotations := gl.Labels()
	assert.Equal(t, 3, len(annotations))
	assert.Equal(t, "gitlab.com", annotations["mondoo.com/exec-environment"])
	assert.Equal(t, "test-job", annotations["gitlab.com/job-name"])
	assert.Equal(t, "testuser", annotations["gitlab.com/user-id"])
}
