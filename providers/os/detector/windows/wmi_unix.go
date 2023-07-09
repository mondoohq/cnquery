//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package windows

import "go.mondoo.com/cnquery/providers/os/connection/shared"

func GetWmiInformation(conn shared.Connection) (*WmicOSInformation, error) {
	return powershellGetWmiInformation(conn)
}
