//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package windows

import "go.mondoo.com/cnquery/providers/os/connection"

func GetWindowsOSBuild(conn connection.Connection) (*WindowsCurrentVersion, error) {
	return powershellGetWindowsOSBuild(conn)
}
