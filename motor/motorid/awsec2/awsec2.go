package awsec2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
)

type InstanceIdentifier interface {
	InstanceID() (string, error)
}

func Resolve(t transports.Transport, p *platform.Platform) (InstanceIdentifier, error) {
	_, ok := t.(*local.LocalTransport)
	if ok {
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "cannot not determine aws environment")
		}
		return NewLocal(cfg), nil
	} else {
		if p.IsFamily(platform.FAMILY_UNIX) || p.IsFamily(platform.FAMILY_WINDOWS) {
			return NewCommandInstanceMetadata(t, p), nil
		}
	}
	return nil, errors.New(fmt.Sprintf("awsec2 id detector is not supported for your asset: %s %s", p.Name, p.Version))
}
