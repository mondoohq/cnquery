package providers

import "errors"

var (
	ErrProviderTypeDoesNotMatch = errors.New("provider type does not match")
	ErrRunCommandNotImplemented = errors.New("provider does not implement RunCommand")
	ErrFileInfoNotImplemented   = errors.New("provider does not implement FileInfo")
)
