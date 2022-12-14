//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package windows

import (
	"go.mondoo.com/cnquery/motor/providers/os"
)

func GetWindowsOSBuild(p os.OperatingSystemProvider) (*WindowsCurrentVersion, error) {
	return powershellGetWindowsOSBuild(p)
}
