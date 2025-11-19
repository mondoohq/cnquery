// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// This file is used to stub out the Darwin and Linux route detection for windows builds
// to avoid compile errors because golang.org/x/sys/unix and golang.org/x/net/route is excluded on Windows.
package networkinterface

import (
	"github.com/cockroachdb/errors"
)

func (n *netr) detectDarwinRoutes() ([]Route, error) {
	return nil, errors.New("Darwin route detection is not available on Windows builds")
}

func (n *netr) detectLinuxRoutes() ([]Route, error) {
	return nil, errors.New("Linux route detection is not available on Windows builds")
}
