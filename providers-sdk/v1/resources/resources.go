package resources

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative resources.proto

// NotReadyError indicates the results are not ready to be processed further
type NotReadyError struct{}

func (n NotReadyError) Error() string {
	return "NotReadyError"
}

type NotFoundError struct {
	Resource string
	ID       string
}

func (n NotFoundError) Error() string {
	return n.Resource + " '" + n.ID + "' not found"
}
