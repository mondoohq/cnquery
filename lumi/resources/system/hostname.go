package system

import (
	"errors"
	"io/ioutil"

	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/platform"
)

func Hostname(motor *motor.Motor) (string, error) {
	var hostname string
	pi, err := motor.Platform()
	if err != nil {
		return hostname, err
	}

	switch {
	case pi.IsFamily(platform.FAMILY_LINUX):
		cmd, err := motor.Transport.RunCommand("hostname")
		if err != nil {
			return hostname, err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return hostname, err
		}
		hostname = string(data)
	case pi.IsFamily(platform.FAMILY_WINDOWS):
		cmd, err := motor.Transport.RunCommand("powershell -c \"$env:computername\"")
		if err != nil {
			return hostname, err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return hostname, err
		}
		hostname = string(data)
	default:
		return hostname, errors.New("your platform is not supported by hostname resource")
	}

	return hostname, nil
}
