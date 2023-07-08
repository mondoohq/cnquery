package platformid

import (
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/providers/os/connection"
)

type UniquePlatformIDProvider interface {
	Name() string
	ID() (string, error)
}

func MachineIDProvider(conn connection.Connection, platform *platform.Platform) (UniquePlatformIDProvider, error) {
	var uuidProvider UniquePlatformIDProvider
	for i := range platform.Family {
		if platform.Family[i] == "linux" {
			uuidProvider = &LinuxIdProvider{connection: conn}
		}
	}

	if uuidProvider == nil && platform.Name == "macos" {
		uuidProvider = &MacOSIdProvider{connection: conn}
	} else if uuidProvider == nil && platform.Name == "windows" {
		uuidProvider = &WinIdProvider{connection: conn}
	}

	return uuidProvider, nil
}
