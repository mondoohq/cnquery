// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared

import (
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

type Connection interface {
	shared.Connection

	// Used to avoid verifying a client with the same options more than once
	Verify() (accountID string, err error)
	Hash() uint64
	SetAccountId(string)
}
