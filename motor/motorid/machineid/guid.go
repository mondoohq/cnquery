package machineid

import (
	"errors"
	"io/ioutil"
	"strings"

	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/platform"
)

func MachineId(motor *motor.Motor) (string, error) {
	var guid string
	pi, err := motor.Platform()
	if err != nil {
		return guid, err
	}

	switch {
	case pi.IsFamily(platform.FAMILY_WINDOWS):
		cmd, err := motor.Transport.RunCommand("powershell -c \"Get-WmiObject Win32_ComputerSystemProduct  | Select-Object -ExpandProperty UUID\"")
		if err != nil {
			return guid, err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return guid, err
		}
		guid = string(data)
	default:
		return guid, errors.New("your platform is not supported by hostname resource")
	}

	return strings.TrimSpace(guid), nil
}
