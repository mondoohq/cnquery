// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"go.mondoo.com/cnquery/v9/motor/providers"
	"go.mondoo.com/cnquery/v9/motor/providers/os"
)

type CommandObservable struct {
	Result *os.Command
}

func (co *CommandObservable) Type() providers.ObservableType {
	return providers.CommandType
}

func (co *CommandObservable) ID() string {
	return co.Result.Command
}

func NewCommandRunnable(command string) func(p os.OperatingSystemProvider) (providers.Observable, error) {
	return func(p os.OperatingSystemProvider) (providers.Observable, error) {
		cmd, err := p.RunCommand(command)
		res := &CommandObservable{Result: cmd}
		return res, err
	}
}
