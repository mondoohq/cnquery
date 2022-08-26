package platformid

import (
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os"
)

func MachineIDProvider(provider os.OperatingSystemProvider, platform *platform.Platform) (UniquePlatformIDProvider, error) {
	var uuidProvider UniquePlatformIDProvider
	for i := range platform.Family {
		if platform.Family[i] == "linux" {
			uuidProvider = &LinuxIdProvider{provider: provider}
		}
	}

	if uuidProvider == nil && platform.Name == "macos" {
		uuidProvider = &MacOSIdProvider{provider: provider}
	} else if uuidProvider == nil && platform.Name == "windows" {
		uuidProvider = &WinIdProvider{provider: provider}
	}

	return uuidProvider, nil
}
