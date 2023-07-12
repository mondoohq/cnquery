package resources

import "errors"

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative resources.proto

// NotReadyError indicates the results are not ready to be processed further
type NotReadyError struct{}

func (n NotReadyError) Error() string {
	return "NotReadyError"
}

var NotFound = errors.New("not found")
