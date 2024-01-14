// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"io"
	"sync"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

type mqlCommandInternal struct {
	lock             sync.Mutex
	commandIsRunning bool
}

func (c *mqlCommand) id() (string, error) {
	return c.Command.Data, c.Command.Error
}

func (c *mqlCommand) execute(cmd string) error {
	c.lock.Lock()
	if c.commandIsRunning {
		c.lock.Unlock()
		return plugin.NotReady
	}
	c.commandIsRunning = true
	c.lock.Unlock()

	x, err := c.MqlRuntime.Connection.(shared.Connection).RunCommand(cmd)
	if err != nil {
		c.Exitcode = plugin.TValue[int64]{Error: err, State: plugin.StateIsSet}
		c.Stdout = plugin.TValue[string]{Error: err, State: plugin.StateIsSet}
		c.Stderr = plugin.TValue[string]{Error: err, State: plugin.StateIsSet}
		return err
	}

	c.Exitcode = plugin.TValue[int64]{Data: int64(x.ExitStatus), State: plugin.StateIsSet}

	stdout, err := io.ReadAll(x.Stdout)
	c.Stdout = plugin.TValue[string]{Data: string(stdout), Error: err, State: plugin.StateIsSet}

	stderr, err := io.ReadAll(x.Stderr)
	c.Stderr = plugin.TValue[string]{Data: string(stderr), Error: err, State: plugin.StateIsSet}

	c.lock.Lock()
	c.commandIsRunning = false
	c.lock.Unlock()

	return nil
}

func (c *mqlCommand) stdout(cmd string) (string, error) {
	// note: we ignore the return value because everything is set in execute
	return "", c.execute(cmd)
}

func (c *mqlCommand) stderr(cmd string) (string, error) {
	// note: we ignore the return value because everything is set in execute
	return "", c.execute(cmd)
}

func (c *mqlCommand) exitcode(cmd string) (int64, error) {
	// note: we ignore the return value because everything is set in execute
	return 0, c.execute(cmd)
}

func runCommand(runtime *plugin.Runtime, cmd string) (string, error) {
	o, err := CreateResource(runtime, "command", map[string]*llx.RawData{
		"command": llx.StringData(cmd),
	})
	if err != nil {
		return "", err
	}

	command := o.(*mqlCommand)
	exit := command.GetExitcode()
	if exit.Error != nil {
		return "", exit.Error
	}
	if exit.Data != 0 {
		err := command.GetStderr()
		if err.Error != nil {
			return "", err.Error
		}
		return "", errors.New(err.Data)
	}

	out := command.GetStdout()
	if out.Error != nil {
		return "", out.Error
	}
	return out.Data, nil
}
