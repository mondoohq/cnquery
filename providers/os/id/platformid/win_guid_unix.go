//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package platformid

import "go.mondoo.com/cnquery/providers/os/connection"

func windowsMachineId(conn connection.Connection) (string, error) {
	return PowershellWindowsMachineId(conn)
}
