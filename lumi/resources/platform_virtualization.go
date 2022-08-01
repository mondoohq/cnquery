package resources

import (
	"go.mondoo.io/mondoo/motor/providers/container/docker_engine"
	"go.mondoo.io/mondoo/motor/providers/container/docker_snapshot"
	"go.mondoo.io/mondoo/motor/providers/tar"
)

func (v *lumiPlatformVirtualization) id() (string, error) {
	return "platform.virtualization", nil
}

func (v *lumiPlatformVirtualization) GetIsContainer() (bool, error) {
	switch v.MotorRuntime.Motor.Transport.(type) {
	case *tar.Transport:
		return true, nil
	case *docker_snapshot.DockerSnapshotTransport:
		return true, nil
	case *docker_engine.Transport:
		return true, nil
	}

	return false, nil
}
