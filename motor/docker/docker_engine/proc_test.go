package docker_engine

// func TestParseProc(t *testing.T) {
// 	trans, err := NewRunningContainer(&types.Target{Backend: "docker", Host: "3746fa27e312"})

// 	f, err := trans.File("/proc/sys/kernel")
// 	if err != nil {
// 		t.Fatalf("cannot request file %v", err)
// 	}

// 	tarReader, err := f.Tar()
// 	if err != nil {
// 		t.Fatalf("cannot request file %v", err)
// 	}

// 	kernelParameters, err := procfs.ParseLinuxSysctl("/proc/sys/", tarReader)
// 	if err != nil {
// 		t.Fatalf("cannot request file %v", err)
// 	}

// 	assert.Equal(t, kernelParameters["kernel.sched_domain.cpu0.domain0.name"], "DIE", "found kernel parameter")
// 	assert.Equal(t, kernelParameters["kernel.sched_domain.cpu1.domain0.flags"], "4143", "found kernel parameter")
// 	assert.Equal(t, kernelParameters["kernel.yama.ptrace_scope"], "1", "found kernel parameter")
// }
