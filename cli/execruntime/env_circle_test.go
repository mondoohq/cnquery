// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package execruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircleCIRuntimeEnv(t *testing.T) {
	// set mock provider
	environmentProvider = newMockEnvProvider()
	require.NoError(t, environmentProvider.Setenv("CI", "1"))
	require.NoError(t, environmentProvider.Setenv("CIRCLECI", "1"))
	require.NoError(t, environmentProvider.Setenv("CIRCLE_REPOSITORY_URL", "https://example.com/project"))
	require.NoError(t, environmentProvider.Setenv("CIRCLE_PROJECT_REPONAME", "example-project"))
	require.NoError(t, environmentProvider.Setenv("CIRCLE_BUILD_NUM", "1"))
	require.NoError(t, environmentProvider.Setenv("CIRCLE_USERNAME", "johndoe"))

	env := Detect()
	assert.True(t, env.IsAutomatedEnv())
	assert.Equal(t, CIRCLE, env.Id)
	assert.Equal(t, "CircleCI", env.Name)

	annotations := env.Labels()
	assert.Equal(t, 5, len(annotations))
	assert.Equal(t, "circleci.com", annotations["mondoo.com/exec-environment"])
	assert.Equal(t, "https://example.com/project", annotations["circleci.com/repository-url"])
	assert.Equal(t, "example-project", annotations["circleci.com/project-reponame"])
	assert.Equal(t, "1", annotations["circleci.com/build-num"])
	assert.Equal(t, "johndoe", annotations["circleci.com/username"])
}
