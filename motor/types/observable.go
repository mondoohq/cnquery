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
