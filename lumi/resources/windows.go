package resources

import (
	"go.mondoo.io/mondoo/lumi/resources/powershell"
	"go.mondoo.io/mondoo/lumi/resources/windows"
)

func (s *lumiWindows) id() (string, error) {
	return "windows", nil
}

func (s *lumiWindows) GetComputerInfo() (map[string]interface{}, error) {

	cmd := windows.PSGetComputerInfo

	// encode the powershell command
	encodedCmd := powershell.Encode(cmd)

	executedCmd, err := s.Runtime.Motor.Transport.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	return windows.ParseComputerInfo(executedCmd.Stdout)
}
