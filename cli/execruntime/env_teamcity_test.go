package execruntime

// func TestTeamcityRuntimeEnv(t *testing.T) {
// 	// set mock provider
// 	environmentProvider = newMockEnvProvider()
// 	environmentProvider.Setenv("CI", "1")
// 	environmentProvider.Setenv("TEAMCITY_PROJECT_NAME", "foo")
// 	environmentProvider.Setenv("BUILD_NUMBER", "123456")

// 	env := Detect()
// 	assert.True(t, env.IsAutomatedEnv())
// 	assert.Equal(t, TEAMCITY, env.Id)
// 	assert.Equal(t, "TeamCity", env.Name)

// 	annotations := env.Labels()
// 	assert.Equal(t, 3, len(annotations))
// 	assert.Equal(t, "jetbrains.com", annotations["mondoo.com/exec-environment"])
// 	assert.Equal(t, "foo", annotations["jetbrains.com/project-name"])
// 	assert.Equal(t, "123456", annotations["jetbrains.com/build-number"])
// }
