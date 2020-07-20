package events

import "go.mondoo.io/mondoo/motor/transports"

type CommandObservable struct {
	Result *transports.Command
}

func (co *CommandObservable) Type() transports.ObservableType {
	return transports.CommandType
}

func (co *CommandObservable) ID() string {
	return co.Result.Command
}

func NewCommandRunnable(command string) func(m transports.Transport) (transports.Observable, error) {
	return func(m transports.Transport) (transports.Observable, error) {
		cmd, err := m.RunCommand(command)
		res := &CommandObservable{Result: cmd}
		return res, err

	}
}
