package machineid

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/id/platformid"
)

func MachineId(conn shared.Connection, p *inventory.Platform) (string, error) {
	uuidProvider, err := platformid.MachineIDProvider(conn, p)
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
