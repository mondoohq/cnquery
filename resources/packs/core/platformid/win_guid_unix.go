//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package platformid

import (
	"go.mondoo.io/mondoo/motor/providers/os"
)

func windowsMachineId(p os.OperatingSystemProvider) (string, error) {
	return PowershellWindowsMachineId(p)
}
