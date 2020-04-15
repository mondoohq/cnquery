package resources

import (
	"io/ioutil"
)

func (c *lumiCommand) id() (string, error) {
	return c.Command()
}

func (c *lumiCommand) execute() (string, string, int, error) {
	cmd, err := c.Command()
	if err != nil {
		return "", "", 1, err
	}

	executedCmd, err := c.Runtime.Motor.Transport.RunCommand(cmd)
	if err != nil {
		return "", "", 1, err
	}

	out, err := ioutil.ReadAll(executedCmd.Stdout)
	if err != nil {
		return "", "", 1, err
	}

	outErr, err := ioutil.ReadAll(executedCmd.Stderr)
	if err != nil {
		return "", "", 1, err
	}

	return string(out), string(outErr), executedCmd.ExitStatus, nil
}

func (c *lumiCommand) GetStdout() (string, error) {
	stdout, _, _, err := c.execute()
	if err != nil {
		return "", err
	}
	return stdout, nil
}

func (c *lumiCommand) GetStderr() (string, error) {
	_, stderr, _, err := c.execute()
	if err != nil {
		return "", err
	}
	return stderr, nil
}

func (c *lumiCommand) GetExitcode() (int64, error) {
	_, _, exitcode, err := c.execute()
	if err != nil {
		return 1, err
	}
	return int64(exitcode), nil
}
