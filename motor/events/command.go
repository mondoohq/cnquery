package events

import (
	"go.mondoo.io/mondoo/motor/types"
)

type CommandObservable struct {
	Result *types.Command
}

func (co *CommandObservable) Type() types.ObservableType {
	return types.CommandType
}

func (co *CommandObservable) ID() string {
	return co.Result.Command
}

func NewCommandRunnable(command string) func(m types.Transport) (types.Observable, error) {
	return func(m types.Transport) (types.Observable, error) {
		cmd, err := m.RunCommand(command)
		res := &CommandObservable{Result: cmd}
		return res, err

	}
}
