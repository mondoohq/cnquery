package vmwareguestapi

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"

	"go.mondoo.io/mondoo/motor/providers/os"
	"go.mondoo.io/mondoo/motor/providers/vmwareguestapi/toolbox"
)

type Command struct {
	os.Command
	tb *toolbox.Client
}

func (c *Command) Exec(command string) (*os.Command, error) {
	c.Command.Command = command
	c.Command.Stats.Duration = time.Since(c.Command.Stats.Start)

	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	c.Command.Stdout = stdoutBuffer
	c.Command.Stderr = stderrBuffer

	if c.tb == nil {
		return nil, errors.New("vmware process manager not set")
	}

	script := "!/bin/sh\n" + c.Command.Command

	ecmd := &exec.Cmd{
		Path:   "",
		Args:   []string{},
		Env:    []string{},
		Dir:    "",
		Stdin:  bytes.NewBuffer([]byte(script)),
		Stdout: stdoutBuffer,
		Stderr: stderrBuffer,
	}

	// start vmware tools call
	ctx := context.Background()
	// TODO: this is not verify efficient for windows since we call powershell via powershell
	// TODO: the toolbox implementation requires bash and we should limit us to /bin/sh
	err := c.tb.Run(ctx, ecmd)
	c.Command.Stats.Duration = time.Since(c.Command.Stats.Start)
	if err != nil {
		// TODO extract exit code, since its a private error, we need to parse the string
		// match := errors.As(err, &e)
		// if match {
		// 	c.Command.ExitStatus = e.ExitStatus()
		// }
		return &c.Command, err
	}
	return &c.Command, nil
}
