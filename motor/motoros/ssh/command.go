package ssh

import (
	"bytes"
	"errors"
	"time"

	"go.mondoo.io/mondoo/motor/motoros/types"
	"golang.org/x/crypto/ssh"
)

type Command struct {
	types.Command
	SSHClient *ssh.Client
}

func (c *Command) Exec(command string) (*types.Command, error) {
	c.Command.Command = command
	c.Command.Stats.Duration = time.Since(c.Command.Stats.Start)

	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	c.Command.Stdout = stdoutBuffer
	c.Command.Stderr = stderrBuffer

	if c.SSHClient == nil {
		return nil, errors.New("ssh session not established")
	}

	session, err := c.SSHClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// start ssh call
	session.Stdout = stdoutBuffer
	session.Stderr = stderrBuffer
	err = session.Run(c.Command.Command)
	c.Command.Stats.Duration = time.Since(c.Command.Stats.Start)
	if err != nil {
		var e *ssh.ExitError
		match := errors.As(err, &e)
		if match {
			c.Command.ExitStatus = e.ExitStatus()
		}
		return &c.Command, err
	}
	return &c.Command, nil
}
