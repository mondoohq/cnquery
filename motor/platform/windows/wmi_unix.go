// +build linux darwin netbsd openbsd freebsd

package windows

import "go.mondoo.io/mondoo/motor/transports"

func GetWmiInformation(t transports.Transport) (*WmicOSInformation, error) {
	return powershellGetWmiInformation(t)
}
