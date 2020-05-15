package types

import (
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/motoros/capabilities"
	"go.mondoo.io/mondoo/nexus/assets"
)

type Transport interface {
	// RunCommand executes a command on the target system
	RunCommand(command string) (*Command, error)
	// File opens a specific file
	File(path string) (afero.File, error)
	// returns file permissions and ownership
	FileInfo(path string) (FileInfoDetails, error)
	// FS provides access to the file system of the target system
	FS() afero.Fs
	// Close closes the transport
	Close()
	// returns if this is a static asset that does not allow run command
	Capabilities() capabilities.Capabilities

	Kind() assets.Kind
	Runtime() string
}
