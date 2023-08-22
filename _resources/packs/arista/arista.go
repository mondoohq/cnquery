// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arista

import "go.mondoo.com/cnquery/resources/packs/arista/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
