package awsec2

import (
	"context"
	"fmt"

	"go.mondoo.com/cnquery/motor/providers/os"

	"errors"
	"github.com/aws/aws-sdk-go-v2/config"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/local"
)

type Identity struct {
	InstanceID   string
	InstanceName string
	AccountID    string
}
type InstanceIdentifier interface {
	Identify() (Identity, error)
}

func Resolve(provider os.OperatingSystemProvider, pf *platform.Platform) (InstanceIdentifier, error) {
	_, ok := provider.(*local.Provider)
	if ok {
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, errors.Join(err, errors.New("cannot not determine aws environment"))
		}
		return NewLocal(cfg), nil
	} else {
		if pf.IsFamily(platform.FAMILY_UNIX) || pf.IsFamily(platform.FAMILY_WINDOWS) {
			// try to fetch a config, even if this is not being ran on the ec2 instance itself.
			cfg, err := config.LoadDefaultConfig(context.Background())
			if err != nil {
				return NewCommandInstanceMetadata(provider, pf, nil), nil
			}
			return NewCommandInstanceMetadata(provider, pf, &cfg), nil
		}
	}
	return nil, errors.New(fmt.Sprintf("awsec2 id detector is not supported for your asset: %s %s", pf.Name, pf.Version))
}
