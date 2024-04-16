// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"go.mondoo.com/cnquery/v11/providers/os/connection/local"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

type LocalCommandRunner struct {
	shell []string
}

func (r *LocalCommandRunner) RunCommand(command string) (*shared.Command, error) {
	c := local.CommandRunner{Shell: r.shell}
	args := []string{}

	res, err := c.Exec(command, args)
	return res, err
}
