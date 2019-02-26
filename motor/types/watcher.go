package types

type ObservableType int

const (
	FileType ObservableType = iota
	CommandType
)

type Observable interface {
	Type() ObservableType
	ID() string
}

type Watcher interface {
	Subscribe(typ string, id string, observable func(Observable)) error
	Unsubscribe(typ string, id string) error
	TearDown() error
}
