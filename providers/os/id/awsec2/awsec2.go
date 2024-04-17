// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package awsec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

type Identity struct {
	InstanceID   string
	InstanceName string
	AccountID    string
}
type InstanceIdentifier interface {
	Identify() (Identity, error)
}

func Resolve(conn shared.Connection, pf *inventory.Platform) (InstanceIdentifier, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		// for local environments we must have a config, or it won't work
		if conn.Type() == shared.Type_Local {
			return nil, errors.Wrap(err, "cannot not determine AWS environment")
		}

		// over a remote connection, we can try without the config
		return NewCommandInstanceMetadata(conn, pf, nil), nil
	}

	if conn.Type() == shared.Type_Local {
		// TODO: Dom: Since a mocked local is not considered local in the original
		// code, we are not testing this code path. Also the original only had
		// mock and non-mock, where the v9 plugin system introduces hybrid modes.
		// We have to revisit this part of the code...
		if _, ok := conn.(*mock.Connection); !ok {
			return NewLocal(cfg), nil
		}
	}
	return NewCommandInstanceMetadata(conn, pf, &cfg), nil
}
