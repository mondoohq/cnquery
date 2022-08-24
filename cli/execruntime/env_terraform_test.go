package execruntime

// func TestDetectTerraform(t *testing.T) {
// 	gl := environmentDef["terraform"]
// 	assert.NotNil(t, gl)
// 	assert.Equal(t, "terraform", gl.Id)
// 	assert.Equal(t, "Terraform", gl.Name)

// 	assert.False(t, gl.Detect())

// 	// set mock provider
// 	environmentProvider = newMockEnvProvider()
// 	environmentProvider.Setenv("CI", "1")
// 	environmentProvider.Setenv("TERRAFORM_PIPELINE", "1")
// 	assert.True(t, gl.Detect())
// 	annotations := gl.Labels()
// 	assert.Equal(t, 1, len(annotations))
// 	assert.Equal(t, "terraform.io", annotations["mondoo.com/exec-environment"])
// }
