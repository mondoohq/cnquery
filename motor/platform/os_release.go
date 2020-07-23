package platform

import (
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports"
)

func NewOSReleaseDetector(t transports.Transport) *OSReleaseDetector {
	return &OSReleaseDetector{
		Transport: t,
	}
}

type OSReleaseDetector struct {
	Transport transports.Transport
}

func (d *OSReleaseDetector) command(command string) (string, error) {
	cmd, err := d.Transport.RunCommand(command)
	if err != nil {
		log.Debug().Err(err).Msg("could not execute os release detection command")
		return "", err
	}

	content, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

// UNIX helper methods
// operating system name
func (d *OSReleaseDetector) unames() (string, error) {
	return d.command("uname -s")
}

// operating system release
func (d *OSReleaseDetector) unamer() (string, error) {
	return d.command("uname -r")
}

// machine hardware name
func (d *OSReleaseDetector) unamem() (string, error) {
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

func (d *OSReleaseDetector) osrelease() (map[string]string, error) {
	f, err := d.Transport.FS().Open("/etc/os-release")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return ParseOsRelease(string(content))
}

// DISTRIB_ID=Ubuntu
// DISTRIB_RELEASE=16.04
// DISTRIB_CODENAME=xenial
// DISTRIB_DESCRIPTION="Ubuntu 16.04.3 LTS"
// lsb release is not the default on newer systems, but can still be used
// as a fallback mechanism
func (d *OSReleaseDetector) lsbconfig() (map[string]string, error) {
	f, err := d.Transport.FS().Open("/etc/lsb-release")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return ParseLsbRelease(string(content))
}

// darwin_swversion will call `/usr/bin/sw_vers` to identify the
// version of darwin. A common output would be:
// ```                                                                                                                  3d master[97c5c29]
// ProductName:	Mac OS X
// ProductVersion:	10.13.2
// BuildVersion:	17C88
// ````
func (d *OSReleaseDetector) darwin_swversion() (map[string]string, error) {
	content, err := d.command("/usr/bin/sw_vers")
	if err != nil {
		return nil, err
	}
	return ParseDarwinRelease(content)
}

// macosSystemVersion is a specifc identifier for the operating system on macos
func (d *OSReleaseDetector) macosSystemVersion() (map[string]string, error) {
	f, err := d.Transport.FS().Open("/System/Library/CoreServices/SystemVersion.plist")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return ParseMacOSSystemVersion(string(content))
}

var majorminor = regexp.MustCompile(`^(\d+)(?:.(\d*)){0,1}(?:.(.*)){0,1}`)

type ReleaseVersion struct {
	Major string
	Minor string
	Other string
}

func (v ReleaseVersion) MajorAtoi() (int, error) {
	return strconv.Atoi(v.Major)
}

func ParseOsVersion(v string) ReleaseVersion {

	m := majorminor.FindStringSubmatch(v)
	if len(m) == 0 {
		return ReleaseVersion{Major: v}
	}

	return ReleaseVersion{Major: m[1], Minor: m[2], Other: m[3]}
}
