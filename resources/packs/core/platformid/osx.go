package platformid

import (
	"errors"
	"io/ioutil"
	"regexp"
	"strings"

	"go.mondoo.com/cnquery/motor/providers/os"
)

// MacOSIdProvider read the operating system id by calling
// "ioreg -rd1 -c IOPlatformExpertDevice" and extracting
// the IOPlatformUUID
type MacOSIdProvider struct {
	provider os.OperatingSystemProvider
}

func (p *MacOSIdProvider) Name() string {
	return "MacOS Platform ID"
}

var MACOS_ID_REGEX = regexp.MustCompile(`\"IOPlatformUUID\"\s*=\s*\"(.*)\"`)

func (p *MacOSIdProvider) ID() (string, error) {
	c, err := p.provider.RunCommand("ioreg -rd1 -c IOPlatformExpertDevice")
	if err != nil || c.ExitStatus != 0 {
		return "", err
	}

	// parse string with regex with \"IOPlatformUUID\"\s*=\s*\"(.*)\"
	content, err := ioutil.ReadAll(c.Stdout)
	if err != nil {
		return "", err
	}

	m := MACOS_ID_REGEX.FindStringSubmatch(string(content))
	if m == nil {
		return "", errors.New("could not detect the machine id")
	}

	return strings.TrimSpace(strings.ToLower(m[1])), nil
}
