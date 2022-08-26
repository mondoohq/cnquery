package os

import (
	"errors"
	"io/ioutil"

	"go.mondoo.com/cnquery/motor/providers/os"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/motor/providers"
)

func (c *mqlCommand) id() (string, error) {
	return c.Command()
}

func (c *mqlCommand) execute() (*os.Command, error) {
	if !c.MotorRuntime.Motor.Provider.Capabilities().HasCapability(providers.Capability_RunCommand) {
		return nil, errors.New("run command not supported on this transport")
	}

	osProvider, err := osProvider(c.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	var executedCmd *os.Command

	cmd, err := c.Command()
	if err != nil {
		return nil, err
	}

	data, ok := c.Cache.Load(cmd)
	if ok {
		executedCmd, ok := data.Data.(*os.Command)
		if ok {
			return executedCmd, data.Error
		}
	}

	// note: we ignore the error here, because we want to give all results
	// (stdout/stderr/exitcode) to the user for handling. otherwise the command
	// resource would be nil and you couldnt do `command('notme').exitcode`
	executedCmd, err = osProvider.RunCommand(cmd)

	c.Cache.Store(cmd, &resources.CacheEntry{Data: executedCmd, Error: err})
	return executedCmd, err
}

func (c *mqlCommand) GetStdout() (string, error) {
	executedCmd, err := c.execute()
	if err != nil {
		return "", err
	}

	out, err := ioutil.ReadAll(executedCmd.Stdout)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (c *mqlCommand) GetStderr() (string, error) {
	executedCmd, err := c.execute()
	if err != nil {
		return "", err
	}

	outErr, err := ioutil.ReadAll(executedCmd.Stderr)
	if err != nil {
		return "", err
	}

	return string(outErr), nil
}

func (c *mqlCommand) GetExitcode() (int64, error) {
	executedCmd, err := c.execute()
	if err != nil {
		return 1, err
	}
	return int64(executedCmd.ExitStatus), nil
}
