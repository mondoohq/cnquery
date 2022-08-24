//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package platformid

import (
	"go.mondoo.com/cnquery/motor/providers/os"
)

func windowsMachineId(p os.OperatingSystemProvider) (string, error) {
	return PowershellWindowsMachineId(p)
}
