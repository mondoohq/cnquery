// +build linux darwin netbsd openbsd freebsd

package platformid

import (
	"go.mondoo.io/mondoo/motor/transports"
)

func windowsMachineId(t transports.Transport) (string, error) {
	return PowershellWindowsMachineId(t)
}
