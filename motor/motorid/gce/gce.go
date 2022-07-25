package gce

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
)

func MondooGcpInstanceID(project string, zone string, instanceID uint64) string {
	return "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/" + project + "/zones/" + zone + "/instances/" + strconv.FormatUint(uint64(instanceID), 10)
}

type InstanceIdentifier interface {
	InstanceID() (string, error)
}

func Resolve(t transports.Transport, p *platform.Platform) (InstanceIdentifier, error) {
	if p.IsFamily(platform.FAMILY_UNIX) || p.IsFamily(platform.FAMILY_WINDOWS) {
		return NewCommandInstanceMetadata(t, p), nil
	}
	return nil, errors.New(fmt.Sprintf("gce id detector is not supported for your asset: %s %s", p.Name, p.Version))
}
