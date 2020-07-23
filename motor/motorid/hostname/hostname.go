package hostname

import (
	"errors"
	"io/ioutil"
	"strings"

	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
)

func Hostname(t transports.Transport, p *platform.Platform) (string, error) {
	var hostname string
	switch {
	case p.IsFamily(platform.FAMILY_UNIX):
		cmd, err := t.RunCommand("hostname")
		if err != nil {
			return hostname, err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return hostname, err
		}

		hostname = string(data)
	case p.IsFamily(platform.FAMILY_WINDOWS):
		cmd, err := t.RunCommand("powershell -c \"$env:computername\"")
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

	return strings.TrimSpace(hostname), nil
}
