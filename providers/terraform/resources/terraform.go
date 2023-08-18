// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/terraform/connection"
)

func terraformConnection(t plugin.Connection) (*connection.Connection, error) {
	gt, ok := t.(*connection.Connection)
	if !ok {
		return nil, errors.New("terraform resource is not supported on this provider")
	}
	return gt, nil
}
