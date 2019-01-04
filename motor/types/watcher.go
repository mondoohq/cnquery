package types

type Watcher interface {
	Subscribe(typ string, id string, observable func(Observable)) error
	Unsubscribe(typ string, id string) error
	TearDown() error
}
