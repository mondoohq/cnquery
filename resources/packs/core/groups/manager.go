package groups

import (
	"errors"

	"go.mondoo.com/cnquery/motor/providers/os"

	"go.mondoo.com/cnquery/motor"
)

func ResolveManager(motor *motor.Motor) (OSGroupManager, error) {
	var gm OSGroupManager

	pf, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	osProvider, isOSProvider := motor.Provider.(os.OperatingSystemProvider)
	if !isOSProvider {
		return nil, errors.New("group manager is not supported for platform: " + pf.Name)
	}

	// check darwin before unix since darwin is also a unix
	if pf.IsFamily("darwin") {
		gm = &OSXGroupManager{provider: osProvider}
	} else if pf.IsFamily("unix") {
		gm = &UnixGroupManager{provider: osProvider}
	} else if pf.IsFamily("windows") {
		gm = &WindowsGroupManager{provider: osProvider}
	}

	if gm == nil {
		return nil, errors.New("could not detect suitable group manager for platform: " + pf.Name)
	}

	return gm, nil
}
