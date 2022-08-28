package execruntime

// func TestDetectPacker(t *testing.T) {
// 	gl := environmentDef["packer"]
// 	assert.NotNil(t, gl)
// 	assert.Equal(t, "packer", gl.Id)
// 	assert.Equal(t, "Packer", gl.Name)

// 	assert.False(t, gl.Detect())

// 	// set mock provider
// 	environmentProvider = newMockEnvProvider()
// 	environmentProvider.Setenv("CI", "1")
// 	environmentProvider.Setenv("PACKER_PIPELINE", "1")
// 	assert.True(t, gl.Detect())
// 	annotations := gl.Labels()
// 	assert.Equal(t, 1, len(annotations))
// 	assert.Equal(t, "packer.io", annotations["mondoo.com/exec-environment"])
// }
