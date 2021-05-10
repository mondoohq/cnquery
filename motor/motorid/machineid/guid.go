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
	var err error
	switch {
	case p.IsFamily(platform.FAMILY_WINDOWS):
		guid, err = windowsMachineId(t)
	default:
		err = errors.New("your platform does not supported by machine-id detection")
	}
	return strings.TrimSpace(guid), err
}

const wmiMachineIDQuery = "SELECT UUID FROM Win32_ComputerSystemProduct"

func powershellWindowsMachineId(t transports.Transport) (string, error) {
	cmd, err := t.RunCommand("powershell -c \"Get-WmiObject -Query '" + wmiMachineIDQuery + "' | Select-Object -ExpandProperty UUID\"")
	if err != nil {
		return "", err
	}
	data, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}
	guid := string(data)
	return guid, nil
}
