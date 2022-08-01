//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package platformid

import (
	"go.mondoo.io/mondoo/motor/providers"
)

func windowsMachineId(t providers.Transport) (string, error) {
	return PowershellWindowsMachineId(t)
}
