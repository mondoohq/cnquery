// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import "go.mondoo.com/cnquery/motor/providers/local"

func IsAzInstalled() bool {
	t, err := local.New()
	if err != nil {
		return false
	}

	command, err := t.RunCommand("az")
	return command.ExitStatus == 0 && err == nil
}
