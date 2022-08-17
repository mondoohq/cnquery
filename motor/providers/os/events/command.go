package events

import "go.mondoo.io/mondoo/motor/providers"

type CommandObservable struct {
	Result *providers.Command
}

func (co *CommandObservable) Type() providers.ObservableType {
	return providers.CommandType
}

func (co *CommandObservable) ID() string {
	return co.Result.Command
}

func NewCommandRunnable(command string) func(m providers.Transport) (providers.Observable, error) {
	return func(m providers.Transport) (providers.Observable, error) {
		cmd, err := m.RunCommand(command)
		res := &CommandObservable{Result: cmd}
		return res, err
	}
}
