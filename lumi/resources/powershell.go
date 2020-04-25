package resources

import (
	"io/ioutil"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/powershell"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

// TODO: consider sharing more code with command resource
func (c *lumiPowershell) id() (string, error) {
	return c.Script()
}

func (c *lumiPowershell) execute() (*types.Command, error) {
	var executedCmd *types.Command

	cmd, err := c.Script()
	if err != nil {
		return nil, err
	}

	// encode the powershell command
	encodedCmd := powershell.Encode(cmd)

	data, ok := c.Cache.Load(encodedCmd)
	if ok {
		executedCmd, ok := data.Data.(*types.Command)
		if ok {
			return executedCmd, nil
		}
	}

	executedCmd, err = c.Runtime.Motor.Transport.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	c.Cache.Store(encodedCmd, &lumi.CacheEntry{Data: executedCmd})
	return executedCmd, nil
}

func (c *lumiPowershell) GetStdout() (string, error) {
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

func (c *lumiPowershell) GetStderr() (string, error) {
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

func (c *lumiPowershell) GetExitcode() (int64, error) {
	executedCmd, err := c.execute()
	if err != nil {
		return 1, err
	}
	return int64(executedCmd.ExitStatus), nil
}
