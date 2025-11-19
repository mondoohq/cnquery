// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// This file is used to stub out the Darwin route detection for Linux builds
package networkinterface

import (
	"github.com/cockroachdb/errors"
)

// Here we are stubbing out the Darwin route detection for Linux builds
// to avoid compile errors because golang.org/x/net/route is excluded on Linux.
func (n *netr) detectDarwinRoutes() ([]Route, error) {
	return nil, errors.New("Darwin route detection is not available on Linux builds")
}
