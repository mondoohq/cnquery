package machineid

import (
	"errors"
	"io/ioutil"
	"strings"

	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
)

func MachineId(t transports.Transport, p *platform.Platform) (string, error) {
	var guid string

	switch {
	case p.IsFamily(platform.FAMILY_WINDOWS):
		cmd, err := t.RunCommand("powershell -c \"Get-WmiObject Win32_ComputerSystemProduct  | Select-Object -ExpandProperty UUID\"")
		if err != nil {
			return guid, err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return guid, err
		}
		guid = string(data)
	default:
		return guid, errors.New("your platform does not supported by machine-id detection")
	}

	return strings.TrimSpace(guid), nil
}
