//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package windows

import "go.mondoo.com/cnquery/providers/os/connection"

func GetWmiInformation(conn connection.Connection) (*WmicOSInformation, error) {
	return powershellGetWmiInformation(conn)
}
