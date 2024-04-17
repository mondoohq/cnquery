// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package execruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJenkinsRuntimeEnv(t *testing.T) {
	// set mock provider
	environmentProvider = newMockEnvProvider()
	require.NoError(t, environmentProvider.Setenv("CI", "1"))
	require.NoError(t, environmentProvider.Setenv("JENKINS_URL", "1"))
	require.NoError(t, environmentProvider.Setenv("GIT_URL", "https://example.com/project"))
	require.NoError(t, environmentProvider.Setenv("JOB_NAME", "example-project"))
	require.NoError(t, environmentProvider.Setenv("BUILD_ID", "1"))
	require.NoError(t, environmentProvider.Setenv("GIT_COMMIT", "12378349271489723489"))

	env := Detect()
	assert.True(t, env.IsAutomatedEnv())
	assert.Equal(t, JENKINS, env.Id)
	assert.Equal(t, "Jenkins CI", env.Name)

	annotations := env.Labels()
	assert.Equal(t, 6, len(annotations))
	assert.Equal(t, "jenkins.io", annotations["mondoo.com/exec-environment"])
	assert.Equal(t, "https://example.com/project", annotations["jenkins.io/giturl"])
	assert.Equal(t, "example-project", annotations["jenkins.io/jobname"])
	assert.Equal(t, "1", annotations["jenkins.io/buildid"])
	assert.Equal(t, "12378349271489723489", annotations["jenkins.io/gitcommit"])
}
