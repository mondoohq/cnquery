package events

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

// FileOp describes a set of file operations.
type FileOp uint32

// These are the generalized file operations that can trigger a notification.
const (
	Create FileOp = 1 << iota
	Write
	Remove
	Rename
	Chmod
	Modify // is this the same of rewrite
	Enoent
	// TODO: distingush between file content and file metadata modify
	Error
)

// file events handling
type FileObservable struct {
	identifier string
	FileOp     FileOp
	File       afero.File
	Error      error
}

func (fo *FileObservable) Type() types.ObservableType {
	return types.FileType
}

func (fo *FileObservable) ID() string {
	return fo.identifier
}

func (fo *FileObservable) Op() FileOp {
	return fo.FileOp
}

func NewFileRunnable(path string) func(m types.Transport) (types.Observable, error) {
	return func(m types.Transport) (types.Observable, error) {
		fileop := Modify
		file, err := m.File(path)

		// TODO: we may want to distingush further, but it does not make sense to do transport specific error handling here
		// therefore we may need common types similar to https://github.com/golang/go/blob/master/src/os/error.go#L22-L23
		if err != nil {
			log.Debug().Err(err).Msg("watch on non-existing file")
			fileop = Error
		}
		return &FileObservable{File: file, FileOp: fileop, Error: err}, nil
	}
}
