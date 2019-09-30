package types

import (
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/motoros/capabilities"
)

type Transport interface {
	// RunCommand executes a command on the target system
	RunCommand(command string) (*Command, error)
	// File opens a specific file
	File(path string) (afero.File, error)
	// FS provides access to the file system of the target system
	FS() afero.Fs
	// Close closes the transport
	Close()
	// returns if this is a static asset that does not allow run command
	Capabilities() []capabilities.Capability
}
