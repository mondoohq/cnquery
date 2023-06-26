package snapshot

import (
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/cmd"
)

type LocalCommandRunner struct {
	shell []string
}

func (r *LocalCommandRunner) RunCommand(command string) (*os.Command, error) {
	c := cmd.CommandRunner{Shell: r.shell}
	args := []string{}

	res, err := c.Exec(command, args)
	return res, err
}
