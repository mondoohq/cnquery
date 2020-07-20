package resources

import (
	"io/ioutil"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/transports"
)

func (c *lumiCommand) id() (string, error) {
	return c.Command()
}

func (c *lumiCommand) execute() (*transports.Command, error) {
	var executedCmd *transports.Command

	cmd, err := c.Command()
	if err != nil {
		return nil, err
	}

	data, ok := c.Cache.Load(cmd)
	if ok {
		executedCmd, ok := data.Data.(*transports.Command)
		if ok {
			return executedCmd, nil
		}
	}

	// note: we ignore the error here, because we want to give all results
	// (stdout/stderr/exitcode) to the user for handling. otherwise the command
	// resource would be nil and you couldnt do `command('notme').exitcode`
	executedCmd, _ = c.Runtime.Motor.Transport.RunCommand(cmd)

	c.Cache.Store(cmd, &lumi.CacheEntry{Data: executedCmd})
	return executedCmd, nil
}

func (c *lumiCommand) GetStdout() (string, error) {
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

func (c *lumiCommand) GetStderr() (string, error) {
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

func (c *lumiCommand) GetExitcode() (int64, error) {
	executedCmd, err := c.execute()
	if err != nil {
		return 1, err
	}
	return int64(executedCmd.ExitStatus), nil
}
