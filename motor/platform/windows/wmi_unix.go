//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package windows

import (
	"go.mondoo.io/mondoo/motor/providers"
)

func GetWmiInformation(t providers.Transport) (*WmicOSInformation, error) {
	return powershellGetWmiInformation(t)
}
