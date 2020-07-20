package groups

import (
	"errors"

	"go.mondoo.io/mondoo/motor"
)

func ResolveManager(motor *motor.Motor) (OSGroupManager, error) {
	var gm OSGroupManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	// check darwin before unix since darwin is also a unix
	if platform.IsFamily("darwin") {
		gm = &OSXGroupManager{motor: motor}
	} else if platform.IsFamily("unix") {
		gm = &UnixGroupManager{motor: motor}
	} else if platform.IsFamily("windows") {
		gm = &WindowsGroupManager{motor: motor}
	}

	if gm == nil {
		return nil, errors.New("could not detect suitable group manager for platform: " + platform.Name)
	}

	return gm, nil
}
