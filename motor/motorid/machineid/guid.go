package machineid

import (
	"errors"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/resources/packs/core/platformid"
)

func MachineId(provider os.OperatingSystemProvider, pf *platform.Platform) (string, error) {
	uuidProvider, err := platformid.MachineIDProvider(provider, pf)
	if err != nil {
		return "", errors.Join(err, errors.New("cannot determine platform uuid"))
	}

	if uuidProvider == nil {
		return "", errors.New("cannot determine platform uuid")
	}

	id, err := uuidProvider.ID()
	if err != nil {
		return "", errors.Join(err, errors.New("cannot determine platform uuid"))
	}

	return id, nil
}
