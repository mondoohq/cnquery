package execruntime

// func TestMonndooRuntimeEnv(t *testing.T) {
// 	gl := environmentDef["mondoo-ci"]
// 	assert.NotNil(t, gl)
// 	assert.Equal(t, "mondoo-ci", gl.Id)
// 	assert.Equal(t, "Mondoo CI", gl.Name)

// 	// set mondoo provider
// 	environmentProvider = newMockEnvProvider()
// 	environmentProvider.Setenv("MONDOO_CI", "1")
// 	environmentProvider.Setenv("CI_COMMIT_SHA", "abc")
// 	environmentProvider.Setenv("CI_BUILD_ID", "1")

// 	env := Detect()
// 	assert.Equal(t, "mondoo-ci", env.Id)
// 	annotations := gl.Labels()
// 	assert.Equal(t, 3, len(annotations))
// 	assert.Equal(t, "ci.mondoo.com", annotations["mondoo.com/exec-environment"])
// }
