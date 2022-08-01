package azcompute

import (
	"fmt"

	"github.com/pkg/errors"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers"
)

type instanceMetadata struct {
	Compute struct {
		ResourceID     string `json:"resourceID"`
		SubscriptionID string `json:"subscriptionId"`
		Tags           string `json:"tags"`
	} `json:"compute"`
}

type InstanceIdentifier interface {
	InstanceID() (string, error)
}

func Resolve(t providers.Transport, p *platform.Platform) (InstanceIdentifier, error) {
	if p.IsFamily(platform.FAMILY_UNIX) || p.IsFamily(platform.FAMILY_WINDOWS) {
		return NewCommandInstanceMetadata(t, p), nil
	}
	return nil, errors.New(fmt.Sprintf("azure compute id detector is not supported for your asset: %s %s", p.Name, p.Version))
}
