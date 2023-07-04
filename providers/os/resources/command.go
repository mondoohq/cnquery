package resources

import (
	"io"
	"sync"

	"go.mondoo.com/cnquery/providers/plugin"
)

type mqlCommandInternal struct {
	lock             sync.Mutex
	commandIsRunning bool
}

// this method is optional, remove it...
func (c *mqlCommand) init(args map[string]interface{}) (map[string]interface{}, *mqlCommand, error) {
	return args, nil, nil
}

func (c *mqlCommand) MqlID() (string, error) {
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

	x, err := c.MqlRuntime.Connection.RunCommand(cmd)
	if err != nil {
		c.Exitcode = plugin.TValue[int64]{Error: err}
		c.Stdout = plugin.TValue[string]{Error: err}
		c.Stderr = plugin.TValue[string]{Error: err}
		return err
	}

	c.Exitcode = plugin.TValue[int64]{Data: int64(x.ExitStatus)}

	stdout, err := io.ReadAll(x.Stdout)
	c.Stdout = plugin.TValue[string]{Data: string(stdout), Error: err}

	stderr, err := io.ReadAll(x.Stderr)
	c.Stderr = plugin.TValue[string]{Data: string(stderr), Error: err}

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
