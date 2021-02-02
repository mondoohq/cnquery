package local

import (
	"bytes"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"go.mondoo.io/mondoo/motor/transports"
)

type Command struct {
	transports.Command
	cmdExecutor *exec.Cmd
	shell       []string
}

func (c *Command) Exec(usercmd string, args []string) (*transports.Command, error) {

	c.Command.Stats.Start = time.Now()

	var cmd string
	cmdArgs := []string{}

	if len(c.shell) > 0 {
		shellCommand, shellArgs := c.shell[0], c.shell[1:]
		cmd = shellCommand
		cmdArgs = append(cmdArgs, shellArgs...)
		cmdArgs = append(cmdArgs, usercmd)
	} else {
		cmd = usercmd
	}
	cmdArgs = append(cmdArgs, args...)

	// this only stores the user command, not the shell
	c.Command.Command = usercmd + " " + strings.Join(args, " ")
	c.cmdExecutor = exec.Command(cmd, cmdArgs...)

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer

	// create buffered stream
	c.Command.Stdout = &stdoutBuffer
	c.Command.Stderr = &stderrBuffer

	c.cmdExecutor.Stdout = c.Command.Stdout
	c.cmdExecutor.Stderr = c.Command.Stderr

	err := c.cmdExecutor.Run()
	c.Command.Stats.Duration = time.Since(c.Command.Stats.Start)

	// command completed successfully, great :-)
	if err == nil {
		return &c.Command, nil
	}

	// if the program failed, we do not return err but its exit code
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			c.Command.ExitStatus = status.ExitStatus()
		}
		return &c.Command, nil
	}

	// all other errors are real errors and not expected
	return &c.Command, err
}
