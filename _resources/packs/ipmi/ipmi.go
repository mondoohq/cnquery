// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ipmi

import "go.mondoo.com/cnquery/resources/packs/ipmi/info"

var Registry = info.Registry

func init() {
	Init(Registry)
}
