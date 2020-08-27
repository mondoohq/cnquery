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

	if !p.IsFamily(platform.FAMILY_UNIX) && !p.IsFamily(platform.FAMILY_WINDOWS) {
		return hostname, errors.New("your platform is not supported by hostname resource")
	}

	// NOTE: hostname command works more reliable than t.RunCommand("powershell -c \"$env:computername\"")
	// since it will return a non-zero exit code.
	cmd, err := t.RunCommand("hostname")
	if err != nil {
		return hostname, err
	}
	data, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return hostname, err
	}

	hostname = string(data)

	return strings.TrimSpace(hostname), nil
}
