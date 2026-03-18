// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package date

import (
	"errors"
	"time"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

type Result struct {
	Time     *time.Time
	Timezone string
}

type Date interface {
	Name() string
	Get() (*Result, error)
}

func New(conn shared.Connection) (Date, error) {
	pf := conn.Asset().Platform

	switch {
	case pf.IsFamily(inventory.FAMILY_UNIX):
		return &Unix{conn: conn}, nil
	case pf.IsFamily(inventory.FAMILY_WINDOWS):
		return &Windows{conn: conn}, nil
	default:
		return nil, errors.New("your platform is not supported by the date resource")
	}
}
