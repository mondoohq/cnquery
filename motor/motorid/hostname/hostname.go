package hostname

import (
	"errors"
	"io/ioutil"
	"strings"

	"go.mondoo.io/mondoo/motor/providers/os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"

	"go.mondoo.io/mondoo/motor/platform"
)

func Hostname(p os.OperatingSystemProvider, pf *platform.Platform) (string, error) {
	var hostname string

	if !pf.IsFamily(platform.FAMILY_UNIX) && !pf.IsFamily(platform.FAMILY_WINDOWS) {
		return hostname, errors.New("your platform is not supported by hostname resource")
	}

	// linux:
	// we prefer the hostname over /etc/hostname since systemd is not updating the value all the time
	//
	// windows:
	// hostname command works more reliable than t.RunCommand("powershell -c \"$env:computername\"")
	// since it will return a non-zero exit code.
	cmd, err := p.RunCommand("hostname")
	if err == nil && cmd.ExitStatus == 0 {
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err == nil {
			return strings.TrimSpace(string(data)), nil
		}
	} else {
		log.Debug().Err(err).Msg("could not run hostname command")
	}

	// try to use /etc/hostname since it's also working on static analysis
	if pf.IsFamily(platform.FAMILY_LINUX) {
		afs := &afero.Afero{Fs: p.FS()}
		ok, err := afs.Exists("/etc/hostname")
		if err == nil && ok {
			content, err := afs.ReadFile("/etc/hostname")
			if err == nil {
				return strings.TrimSpace(string(content)), nil
			}
		} else {
			log.Debug().Err(err).Msg("could not read /etc/hostname file")
		}
	}

	return "", errors.New("could not detect hostname")
}
