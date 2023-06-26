package gce

import (
	"fmt"
	"strconv"

	"go.mondoo.com/cnquery/motor/providers/os"

	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/motor/platform"
)

// deprecated: use MondooGcpInstancePlatformMrn
func MondooGcpInstanceID(project string, zone string, instanceID uint64) string {
	return "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/" + project + "/zones/" + zone + "/instances/" + strconv.FormatUint(uint64(instanceID), 10)
}

func MondooGcpInstancePlatformMrn(project string, zone string, instanceName string) string {
	return "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/" + project + "/zones/" + zone + "/instances/" + instanceName
}

type Identity struct {
	ProjectID string
	// deprecated: use PlatformMrn
	InstanceID  string
	PlatformMrn string
}
type InstanceIdentifier interface {
	Identify() (Identity, error)
}

func Resolve(provider os.OperatingSystemProvider, pf *platform.Platform) (InstanceIdentifier, error) {
	if pf.IsFamily(platform.FAMILY_UNIX) || pf.IsFamily(platform.FAMILY_WINDOWS) {
		return NewCommandInstanceMetadata(provider, pf), nil
	}
	return nil, errors.New(fmt.Sprintf("gce id detector is not supported for your asset: %s %s", pf.Name, pf.Version))
}
