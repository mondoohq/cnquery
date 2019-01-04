package types

type Transport interface {
	RunCommand(command string) (*Command, error)
	File(path string) (File, error)
	Close()
}
