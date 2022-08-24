package awsec2

import (
	"context"
	"fmt"

	"go.mondoo.com/cnquery/motor/providers/os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/local"
)

type InstanceIdentifier interface {
	InstanceID() (string, error)
}

func Resolve(provider os.OperatingSystemProvider, pf *platform.Platform) (InstanceIdentifier, error) {
	_, ok := provider.(*local.Provider)
	if ok {
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "cannot not determine aws environment")
		}
		return NewLocal(cfg), nil
	} else {
		if pf.IsFamily(platform.FAMILY_UNIX) || pf.IsFamily(platform.FAMILY_WINDOWS) {
			return NewCommandInstanceMetadata(provider, pf), nil
		}
	}
	return nil, errors.New(fmt.Sprintf("awsec2 id detector is not supported for your asset: %s %s", pf.Name, pf.Version))
}
