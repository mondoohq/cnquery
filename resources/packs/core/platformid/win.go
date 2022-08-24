package platformid

import (
	"io/ioutil"

	"go.mondoo.com/cnquery/motor/providers/os"
)

const wmiMachineIDQuery = "SELECT UUID FROM Win32_ComputerSystemProduct"

func PowershellWindowsMachineId(p os.OperatingSystemProvider) (string, error) {
	cmd, err := p.RunCommand("powershell -c \"Get-WmiObject -Query '" + wmiMachineIDQuery + "' | Select-Object -ExpandProperty UUID\"")
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
	provider os.OperatingSystemProvider
}

func (p *WinIdProvider) Name() string {
	return "Windows Machine ID"
}

func (p *WinIdProvider) ID() (string, error) {
	return windowsMachineId(p.provider)
}
