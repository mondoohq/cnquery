// +build linux darwin netbsd openbsd freebsd

package machineid

import "go.mondoo.io/mondoo/motor/transports"

func windowsMachineId(t transports.Transport) (string, error) {
	return powershellWindowsMachineId(t)
}
