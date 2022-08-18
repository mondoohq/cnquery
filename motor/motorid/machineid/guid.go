package machineid

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers/os"
	"go.mondoo.io/mondoo/resources/packs/core/platformid"
)

func MachineId(provider os.OperatingSystemProvider, pf *platform.Platform) (string, error) {
	uuidProvider, err := platformid.MachineIDProvider(provider, pf)
	if err != nil {
		return "", errors.Wrap(err, "cannot determine platform uuid")
	}

	if uuidProvider == nil {
		return "", errors.New("cannot determine platform uuid")
	}

	id, err := uuidProvider.ID()
	if err != nil {
		return "", errors.Wrap(err, "cannot determine platform uuid")
	}

	return id, nil
}
