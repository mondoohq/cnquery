package resources

import (
	"go.mondoo.io/mondoo/motor/transports/container/docker_engine"
	"go.mondoo.io/mondoo/motor/transports/container/docker_snapshot"
	"go.mondoo.io/mondoo/motor/transports/tar"
)

func (v *lumiPlatformVirtualization) id() (string, error) {
	return "platform.virtualization", nil
}

func (v *lumiPlatformVirtualization) GetIsContainer() (bool, error) {
	switch v.Runtime.Motor.Transport.(type) {
	case *tar.Transport:
		return true, nil
	case *docker_snapshot.DockerSnapshotTransport:
		return true, nil
	case *docker_engine.Transport:
		return true, nil
	}

	return false, nil
}
