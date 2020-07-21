package transports

//go:generate protoc --proto_path=$GOPATH/src:. --proto_path=$GOPATH/pkg/mod/github.com/gogo/protobuf@v1.3.1/gogoproto --falcon_out=. --iam-actions_out=. --gofast_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types:$GOPATH/src transports.proto

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
