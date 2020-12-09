package resources

import (
	"go.mondoo.io/mondoo/motor/transports/docker/docker_engine"
	"go.mondoo.io/mondoo/motor/transports/docker/image"
	"go.mondoo.io/mondoo/motor/transports/docker/snapshot"
)

func (v *lumiPlatformVirtualization) id() (string, error) {
	return "platform.virtualization", nil
}

func (v *lumiPlatformVirtualization) GetIsContainer() (bool, error) {
	switch v.Runtime.Motor.Transport.(type) {
	case *image.DockerImageTransport:
		return true, nil
	case *snapshot.DockerSnapshotTransport:
		return true, nil
	case *docker_engine.Transport:
		return true, nil
	}

	return false, nil
}
