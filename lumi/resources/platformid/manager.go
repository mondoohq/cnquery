package platformid

import (
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers"
)

func MachineIDProvider(t providers.Transport, p *platform.Platform) (UniquePlatformIDProvider, error) {
	var uuidProvider UniquePlatformIDProvider
	for i := range p.Family {
		if p.Family[i] == "linux" {
			uuidProvider = &LinuxIdProvider{Transport: t}
		}
	}

	if uuidProvider == nil && p.Name == "macos" {
		uuidProvider = &MacOSIdProvider{Transport: t}
	} else if uuidProvider == nil && p.Name == "windows" {
		uuidProvider = &WinIdProvider{Transport: t}
	}

	return uuidProvider, nil
}
