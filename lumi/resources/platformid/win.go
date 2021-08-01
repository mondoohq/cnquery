package platformid

import (
	"io/ioutil"

	"go.mondoo.io/mondoo/motor/transports"
)

const wmiMachineIDQuery = "SELECT UUID FROM Win32_ComputerSystemProduct"

func PowershellWindowsMachineId(t transports.Transport) (string, error) {
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

type WinIdProvider struct {
	Transport transports.Transport
}

func (p *WinIdProvider) Name() string {
	return "Windows Machine ID"
}

func (p *WinIdProvider) ID() (string, error) {
	return windowsMachineId(p.Transport)
}
