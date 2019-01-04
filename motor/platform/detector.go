package platform

import (
	"io/ioutil"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/motorutil"
	"go.mondoo.io/mondoo/motor/parser"
	"go.mondoo.io/mondoo/motor/types"
)

type Info struct {
	Name    string   `json:"name"`
	Title   string   `json:"title"`
	Family  []string `json:"family"`
	Release string   `json:"release"`
	Arch    string   `json:"arch"`
}

func (di *Info) IsFamily(family string) bool {
	for _, entry := range di.Family {
		if entry == family {
			return true
		}
	}
	return false
}

type Detector struct {
	Transport types.Transport
}

func (d *Detector) Resolve() (bool, *Info) {
	log.Debug().Msgf("detector> build up platfrom tree")
	platforms, _ := d.buildPlatformTree()
	log.Debug().Msg("detector> start resolving the platfrom")
	return platforms.Resolve()
}

func (d *Detector) command(command string) (string, error) {
	cmd, err := d.Transport.RunCommand(command)
	if err != nil {
		log.Debug().Err(err)
	}

	content, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

// UNIX helper methods
// operating system name
func (d *Detector) unames() (string, error) {
	return d.command("uname -s")
}

// operating system release
func (d *Detector) unamer() (string, error) {
	return d.command("uname -r")
}

// machine hardware name
func (d *Detector) unamem() (string, error) {
	return d.command("uname -m")
}

// Linux Helper Methods

// NAME="Ubuntu"
// VERSION="16.04.3 LTS (Xenial Xerus)"
// ID=ubuntu
// ID_LIKE=debian
// PRETTY_NAME="Ubuntu 16.04.3 LTS"
// VERSION_ID="16.04"
// HOME_URL="http://www.ubuntu.com/"
// SUPPORT_URL="http://help.ubuntu.com/"
// BUG_REPORT_URL="http://bugs.launchpad.net/ubuntu/"
// VERSION_CODENAME=xenial
// UBUNTU_CODENAME=xenial

func (d *Detector) osrelease() (map[string]string, error) {
	f, err := d.Transport.File("/etc/os-release")
	if err != nil {
		return nil, err
	}

	content, err := motorutil.ReadFile(f)
	if err != nil {
		return nil, err
	}

	return parser.ParseOsRelease(string(content))
}

// DISTRIB_ID=Ubuntu
// DISTRIB_RELEASE=16.04
// DISTRIB_CODENAME=xenial
// DISTRIB_DESCRIPTION="Ubuntu 16.04.3 LTS"
// lsb release is not the default on newer systems, but can still be used
// as a fallback mechanism
func (d *Detector) lsbconfig() (map[string]string, error) {
	f, err := d.Transport.File("/etc/lsb-release")
	if err != nil {
		return nil, err
	}

	content, err := motorutil.ReadFile(f)
	if err != nil {
		return nil, err
	}

	return parser.ParseLsbRelease(string(content))
}

// darwin_swversion will call `/usr/bin/sw_vers` to identify the
// version of darwin. A common output would be:
// ```                                                                                                                  3d master[97c5c29]
// ProductName:	Mac OS X
// ProductVersion:	10.13.2
// BuildVersion:	17C88
// ````
func (d *Detector) darwin_swversion() (map[string]string, error) {
	content, err := d.command("/usr/bin/sw_vers")
	if err != nil {
		return nil, err
	}
	return parser.ParseDarwinRelease(content)
}

// macosSystemVersion is a specifc identifier for the operating system on macos
func (d *Detector) macosSystemVersion() (map[string]string, error) {
	f, err := d.Transport.File("/System/Library/CoreServices/SystemVersion.plist")
	if err != nil {
		return nil, err
	}

	content, err := motorutil.ReadFile(f)
	if err != nil {
		return nil, err
	}

	return parser.ParseMacOSSystemVersion(string(content))
}
