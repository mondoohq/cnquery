//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package windows

import "go.mondoo.io/mondoo/motor/providers"

func GetWindowsOSBuild(t providers.Transport) (*WindowsCurrentVersion, error) {
	return powershellGetWindowsOSBuild(t)
}
