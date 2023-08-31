// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"go.mondoo.com/cnquery/providers/os/connection"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
)

type LocalCommandRunner struct {
	shell []string
}

func (r *LocalCommandRunner) RunCommand(command string) (*shared.Command, error) {
	c := connection.CommandRunner{Shell: r.shell}
	args := []string{}

	res, err := c.Exec(command, args)
	return res, err
}
