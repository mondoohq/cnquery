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
	Eonet
	// TODO: distingush between file content and file metadata modify
)

// file events handling
type FileObservable struct {
	identifier string
	FileOp     FileOp
	File       afero.File
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
		if err != nil {
			log.Debug().Err(err).Msg("watch on non-existing file")
			fileop = Eonet
		}
		return &FileObservable{File: file, FileOp: fileop}, nil
	}
}
