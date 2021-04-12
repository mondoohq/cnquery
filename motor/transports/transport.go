package transports

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --falcon_out=. --iam-actions_out=. transports.proto

import (
	"regexp"

	"github.com/spf13/afero"
)

type Transport interface {
	// RunCommand executes a command on the target system
	RunCommand(command string) (*Command, error)
	// returns file permissions and ownership
	FileInfo(path string) (FileInfoDetails, error)
	// FS provides access to the file system of the target system
	FS() afero.Fs
	// Close closes the transport
	Close()
	// returns if this is a static asset that does not allow run command
	Capabilities() Capabilities

	Kind() Kind
	Runtime() string
}

type FileSearch interface {
	Find(from string, r *regexp.Regexp, typ string) ([]string, error)
}
