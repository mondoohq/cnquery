// +build linux darwin netbsd openbsd freebsd

package win

import "go.mondoo.io/mondoo/motor/transports"

func GetWindowsOSBuild(t transports.Transport) (*WindowsCurrentVersion, error) {
	return powershellGetWindowsOSBuild(t)
}
