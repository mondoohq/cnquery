// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
//
// This file is auto-generated by 'make providers/config'
// and configured via 'providers.yaml'; DO NOT EDIT.

package providers

import (
	_ "embed"
	// osconf "go.mondoo.com/cnquery/v11/providers/os/config"
	// os "go.mondoo.com/cnquery/v11/providers/os/provider"
)

// //go:embed os/resources/os.resources.json
// var osInfo []byte

func init() {
	// builtinProviders[osconf.Config.ID] = &builtinProvider{
	// 	Runtime: &RunningProvider{
	// 		Name:     osconf.Config.Name,
	// 		ID:       osconf.Config.ID,
	// 		Plugin:   os.Init(),
	// 		Schema:   MustLoadSchema("os", osInfo),
	// 		isClosed: false,
	// 	},
	// 	Config: &osconf.Config,
	// }

}
