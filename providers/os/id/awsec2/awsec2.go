package awsec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/providers/os/connection"
)

type Identity struct {
	InstanceID   string
	InstanceName string
	AccountID    string
}
type InstanceIdentifier interface {
	Identify() (Identity, error)
}

func Resolve(conn connection.Connection, pf *platform.Platform) (InstanceIdentifier, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		// for local environments we must have a config, or it won't work
		if conn.Type() == connection.Local {
			return nil, errors.Wrap(err, "cannot not determine AWS environment")
		}

		// over a remote connection, we can try without the config
		return NewCommandInstanceMetadata(conn, pf, nil), nil
	}

	if conn.Type() == connection.Local {
		return NewLocal(cfg), nil
	}
	return NewCommandInstanceMetadata(conn, pf, &cfg), nil
}
