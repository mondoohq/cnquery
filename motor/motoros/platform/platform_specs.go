package platform

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

// often used family names
var (
	FAMILY_UNIX    = "unix"
	FAMILY_DARWIN  = "darwin"
	FAMILY_LINUX   = "linux"
	FAMILY_WINDOWS = "windows"
)

// Operating Systems
var macOS = &PlatformResolver{
	Name:    "macos",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		// when we reach here, we know it is darwin
		// check xml /System/Library/CoreServices/SystemVersion.plist
		f, err := t.File("/System/Library/CoreServices/SystemVersion.plist")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := ioutil.ReadAll(f)
		if err != nil || len(c) == 0 {
			return false, nil
		}

		sv, err := ParseMacOSSystemVersion(string(c))
		if err != nil || len(c) == 0 {
			return false, nil
		}

		di.Name = "mac_os_x"
		di.Title = sv["ProductName"]
		di.Release = sv["ProductVersion"]

		return true, nil
	},
}

// is part of the darwin platfrom and fallback for non-known darwin systems
var otherDarwin = &PlatformResolver{
	Name:    "darwin",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		return true, nil
	},
}

var alpine = &PlatformResolver{
	Name:    "alpine",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if di.Name == "alpine" {
			return true, nil
		}

		f, err := t.File("/etc/alpine-release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := ioutil.ReadAll(f)
		if err != nil || len(c) == 0 {
			return false, nil
		}

		di.Name = "alpine"
		return true, nil
	},
}

var arch = &PlatformResolver{
	Name:    "arch",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if di.Name == "arch" {
			return true, nil
		}
		return false, nil
	},
}

var manjaro = &PlatformResolver{
	Name:    "manjaro",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if di.Name == "manjaro" {
			return true, nil
		}
		return false, nil
	},
}

var debian = &PlatformResolver{
	Name:    "debian",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		osrd := NewOSReleaseDetector(t)

		f, err := t.File("/etc/debian_version")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := ioutil.ReadAll(f)
		if err != nil || len(c) == 0 {
			return false, nil
		}

		osr, err := osrd.osrelease()
		if err != nil {
			return false, nil
		}

		if osr["ID"] != "debian" {
			return false, nil
		}

		di.Title = osr["NAME"]
		di.Release = strings.TrimSpace(string(c))

		unamem, err := osrd.unamem()
		if err == nil {
			di.Arch = unamem
		}

		return true, nil
	},
}

var ubuntu = &PlatformResolver{
	Name:    "ubuntu",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if di.Name == "ubuntu" {
			return true, nil
		}
		return false, nil
	},
}

var raspbian = &PlatformResolver{
	Name:    "raspbian",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if di.Name == "raspbian" {
			return true, nil
		}
		return false, nil
	},
}

var kali = &PlatformResolver{
	Name:    "kali",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if di.Name == "kali" {
			return true, nil
		}
		return false, nil
	},
}

var rhel = &PlatformResolver{
	Name:    "redhat",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		// etc redhat release was parsed by the family already,
		// we reuse that information here
		// e.g. Red Hat Linux, Red Hat Enterprise Linux Server
		if strings.Contains(di.Title, "Red Hat") || di.Name == "redhat" {
			di.Name = "redhat"
			return true, nil
		}

		return false, nil
	},
}

var centos = &PlatformResolver{
	Name:    "centos",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		// works for centos 5+
		if strings.Contains(di.Title, "CentOS") || di.Name == "centos" {
			di.Name = "centos"
			return true, nil
		}

		// CentOS 5 does not have /etc/centos-release
		// check if we have /etc/centos-release file
		f, err := t.File("/etc/centos-release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := ioutil.ReadAll(f)
		if err != nil || len(c) == 0 {
			return false, nil
		}

		// if len(di.Name) == 0 {
		// 	di.Name = "centos"
		// }

		return true, nil
	},
}

var fedora = &PlatformResolver{
	Name:    "fedora",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if strings.Contains(di.Title, "Fedora") || di.Name == "fedora" {
			di.Name = "fedora"
			return true, nil
		}

		// check if we have /etc/fedora-release file
		f, err := t.File("/etc/fedora-release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := ioutil.ReadAll(f)
		if err != nil || len(c) == 0 {
			return false, nil
		}

		// if len(di.Name) == 0 {
		// 	di.Name = "fedora"
		// }

		return true, nil
	},
}

var oracle = &PlatformResolver{
	Name:    "oracle",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		// works for oracle 7+
		if di.Name == "ol" {
			return true, nil
		}

		// check if we have /etc/centos-release file
		f, err := t.File("/etc/oracle-release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := ioutil.ReadAll(f)
		if err != nil || len(c) == 0 {
			return false, nil
		}

		if len(di.Name) == 0 {
			di.Name = "ol"
		}

		return true, nil
	},
}

var scientific = &PlatformResolver{
	Name:    "scientific",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		// works for oracle 7+
		if di.Name == "scientific" {
			return true, nil
		}

		// we only get here if this is a rhel distribution
		if strings.Contains(di.Title, "Scientific Linux") {
			di.Name = "scientific"
			return true, nil
		}

		return false, nil
	},
}

var amazonlinux = &PlatformResolver{
	Name:    "amzn",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if di.Name == "amzn" {
			return true, nil
		}
		return false, nil
	},
}
var windriver = &PlatformResolver{
	Name:    "wrlinux",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if di.Name == "wrlinux" {
			return true, nil
		}
		return false, nil
	},
}

var opensuse = &PlatformResolver{
	Name:    "opensuse",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if di.Name == "opensuse" || di.Name == "opensuse-leap" || di.Name == "opensuse-tumbleweed" {
			return true, nil
		}

		return false, nil
	},
}

var sles = &PlatformResolver{
	Name:    "sles",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if di.Name == "sles" {
			return true, nil
		}
		return false, nil
	},
}

var gentoo = &PlatformResolver{
	Name:    "gentoo",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		f, err := t.File("/etc/gentoo-release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := ioutil.ReadAll(f)
		if err != nil || len(c) == 0 {
			log.Debug().Err(err)
			return false, nil
		}

		content := strings.TrimSpace(string(c))
		name, release, err := ParseRhelVersion(content)
		if err == nil {
			// only set title if not already properly detected by lsb or os-release
			if len(di.Title) == 0 {
				di.Title = name
			}
			if len(di.Release) == 0 {
				di.Release = release
			}
		}

		return false, nil
	},
}

var busybox = &PlatformResolver{
	Name:    "busybox",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {

		command := "ls --help 2>&1 | head -1"
		cmd, err := t.RunCommand(command)
		if err != nil {
			return false, nil
		}
		busy_info, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return false, err
		}

		r := regexp.MustCompile(`^\s*(.*)\s(v[\d\.]+)\s*\((.*)\s*$`)
		m := r.FindStringSubmatch(string(busy_info))
		if len(m) >= 2 {
			title := m[1]
			release := m[2]

			if strings.ToLower(title) == "busybox" {
				di.Name = "busybox"
				di.Title = title
				di.Release = release
				return true, nil
			}
		}

		return false, nil
	},
}

var photon = &PlatformResolver{
	Name:    "photon",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if di.Name == "photon" {
			return true, nil
		}
		return false, nil
	},
}

var openwrt = &PlatformResolver{
	Name:    "openwrt",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		// No clue why they are not using either lsb-release or os-release
		f, err := t.File("/etc/openwrt_release")
		if err != nil {
			return false, err
		}
		defer f.Close()

		content, err := ioutil.ReadAll(f)
		if err != nil {
			return false, err
		}

		lsb, err := ParseLsbRelease(string(content))
		if err == nil {
			if len(lsb["DISTRIB_ID"]) > 0 {
				di.Name = strings.ToLower(lsb["DISTRIB_ID"])
				di.Title = lsb["DISTRIB_ID"]
			}
			if len(lsb["DISTRIB_RELEASE"]) > 0 {
				di.Release = lsb["DISTRIB_RELEASE"]
			}

			return true, nil
		}

		return false, nil
	},
}

// fallback linux detection, since we do not know the system, the family detection may not be correct
var defaultLinux = &PlatformResolver{
	Name:    "generic-linux",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		// if we reach here, we know that we detected linux already
		log.Debug().Msg("platform> we do not know the linux system, but we do our best in guessing")
		return true, nil
	},
}

var netbsd = &PlatformResolver{
	Name:    "netbsd",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if strings.Contains(strings.ToLower(di.Name), "netbsd") == false {
			return false, nil
		}

		osrd := NewOSReleaseDetector(t)
		r, err := osrd.unamer()
		if err == nil {
			di.Release = r
		}

		return true, nil
	},
}

var freebsd = &PlatformResolver{
	Name:    "freebsd",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if strings.Contains(strings.ToLower(di.Name), "freebsd") == false {
			return false, nil
		}

		osrd := NewOSReleaseDetector(t)
		r, err := osrd.unamer()
		if err == nil {
			di.Release = r
		}

		return true, nil
	},
}

var windows = &PlatformResolver{
	Name:    "windows",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		// wmic is available since Windows Server 2008/Vista
		command := "wmic os get * /format:csv"
		cmd, err := t.RunCommand(command)
		if err != nil {
			return false, nil
		}

		data, err := ParseWinWmicOS(cmd.Stdout)
		if err != nil {
			return false, nil
		}

		di.Name = "windows"
		di.Title = data.Caption

		// instead of using windows major.minor.build.ubr we just use build.ubr since
		// major and minor can be derived from the build version
		di.Release = data.BuildNumber

		// FIXME: we need to ask wmic cpu get architecture
		di.Arch = data.OSArchitecture

		// optional: try to get the ubr number (win 10 + 2019)
		pscommand := "Get-ItemProperty -Path 'HKLM:\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion' -Name CurrentBuild, UBR, EditionID | ConvertTo-Json"
		cmd, err = t.RunCommand(fmt.Sprintf("powershell -c \"%s\"", pscommand))
		if err == nil {
			current, err := ParseWinRegistryCurrentVersion(cmd.Stdout)
			if err == nil && current.UBR > 0 {
				di.Release = fmt.Sprintf("%s.%d", di.Release, current.UBR)
			} else {
				log.Debug().Err(err).Msg("could not parse windows current version")
			}
		}

		return true, nil
	},
}

// Families
var darwinFamily = &PlatformResolver{
	Name:     FAMILY_DARWIN,
	Familiy:  true,
	Children: []*PlatformResolver{macOS, otherDarwin},
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		if strings.Contains(strings.ToLower(di.Name), "darwin") == false {
			return false, nil
		}
		// from here we know it is a darwin system

		// read information from /usr/bin/sw_vers
		osrd := NewOSReleaseDetector(t)
		dsv, err := osrd.darwin_swversion()
		// ignore dsv config if we got an error
		if err == nil {
			if len(dsv["ProductName"]) > 0 {
				// TODO: name needs to be slugged
				di.Name = strings.ToLower(dsv["ProductName"])
				di.Title = dsv["ProductName"]
			}
			if len(dsv["ProductVersion"]) > 0 {
				di.Release = dsv["ProductVersion"]
			}
		} else {
			// TODO: we know its darwin, but without swversion support
			log.Error().Err(err)
		}

		return true, nil
	},
}

var bsdFamily = &PlatformResolver{
	Name:     "bsd",
	Familiy:  true,
	Children: []*PlatformResolver{darwinFamily, netbsd, freebsd},
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		osrd := NewOSReleaseDetector(t)
		unames, err := osrd.unames()
		if err != nil {
			return false, err
		}

		unamem, err := osrd.unamem()
		if err == nil {
			di.Arch = unamem
		}

		if len(unames) > 0 {
			di.Name = strings.ToLower(unames)
			di.Title = unames
			return true, nil
		}
		return false, nil
	},
}

var redhatFamily = &PlatformResolver{
	Name:     "redhat",
	Familiy:  true,
	Children: []*PlatformResolver{rhel, centos, fedora, oracle, scientific},
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		f, err := t.File("/etc/redhat-release")
		if err != nil {
			log.Debug().Err(err)
			return false, nil
		}
		defer f.Close()

		c, err := ioutil.ReadAll(f)
		if err != nil || len(c) == 0 {
			log.Debug().Err(err)
			return false, nil
		}

		content := strings.TrimSpace(string(c))
		title, release, err := ParseRhelVersion(content)
		if err == nil {
			log.Debug().Str("title", title).Str("release", release).Msg("detected rhelish platform")
			// only set title if not already properly detected by lsb or os-release
			if len(di.Title) == 0 {
				di.Title = title
			}

			// always override the version from the release file, since it is
			// more accurate
			if len(release) > 0 {
				di.Release = release
			}
		}

		return true, nil
	},
}

var debianFamily = &PlatformResolver{
	Name:     "debian",
	Familiy:  true,
	Children: []*PlatformResolver{debian, ubuntu, raspbian, kali},
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		return true, nil
	},
}

var suseFamily = &PlatformResolver{
	Name:     "suse",
	Familiy:  true,
	Children: []*PlatformResolver{opensuse, sles},
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		return true, nil
	},
}

var archFamily = &PlatformResolver{
	Name:     "arch",
	Familiy:  true,
	Children: []*PlatformResolver{arch, manjaro},
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		// if the file exists, we are on arch or one of its derivates
		f, err := t.File("/etc/arch-release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := ioutil.ReadAll(f)
		if err != nil {
			return false, nil
		}

		// on arch containers, /etc/os-release may not be present
		if len(di.Name) == 0 && strings.Contains(strings.ToLower(string(c)), "manjaro") {
			di.Name = "manjaro"
			di.Title = strings.TrimSpace(string(c))
			return true, nil
		}

		if len(di.Name) == 0 {
			// fallback to arch
			di.Name = "arch"
			di.Title = "Arch Linux"
		}
		return true, nil
	},
}

var linuxFamily = &PlatformResolver{
	Name:     FAMILY_LINUX,
	Familiy:  true,
	Children: []*PlatformResolver{archFamily, redhatFamily, debianFamily, suseFamily, amazonlinux, alpine, gentoo, busybox, photon, windriver, openwrt, defaultLinux},
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		detected := false
		osrd := NewOSReleaseDetector(t)

		lsb, err := osrd.lsbconfig()
		// ignore lsb config if we got an error
		if err == nil {
			if len(lsb["DISTRIB_ID"]) > 0 {
				di.Name = strings.ToLower(lsb["DISTRIB_ID"])
				di.Title = lsb["DISTRIB_ID"]
			}
			if len(lsb["DISTRIB_RELEASE"]) > 0 {
				di.Release = lsb["DISTRIB_RELEASE"]
			}

			detected = true
		} else {
			log.Debug().Err(err).Msg("platform> cannot parse lsb config on this linux system")
		}

		osr, err := osrd.osrelease()
		// ignore os release if we have an error
		if err != nil {
			log.Debug().Err(err).Msg("platform> cannot parse os-release on this linux system")
		} else {
			if len(osr["ID"]) > 0 {
				di.Name = osr["ID"]
			}
			if len(osr["NAME"]) > 0 {
				di.Title = osr["NAME"]
			}
			if len(osr["VERSION_ID"]) > 0 {
				di.Release = osr["VERSION_ID"]
			}

			detected = true
		}

		// Centos 6 does not include /etc/os-release or /etc/lsb-release, therefore any static analysis
		// will not be able to detect the system, since the following unamem and unames mechanism is not
		// available there. Instead the system can be identified by the availability of /etc/redhat-release
		// If /etc/redhat-release is available, we know its a linux system.
		f, err := t.File("/etc/redhat-release")
		if f != nil {
			f.Close()
		}

		if err == nil {
			detected = true
		}

		// try to read the architecture, we cannot assume this works if we use the tar bakcend where we
		// just load the filesystem, therefore we do not fail here
		unamem, err := osrd.unamem()
		if err == nil {
			di.Arch = unamem
		}

		// abort if os-release pr lsb config was available, we don't need uname -s then
		if detected == true {
			return true, nil
		}

		// if we reached here, we have a strange linux distro because it does not ship with
		// lsb config and/or os release information, lets use the uname test to verify that this
		// is a linux, it will fail for container images without the ability to run a process
		unames, err := osrd.unames()
		if err != nil {
			return false, err
		}

		if strings.Contains(strings.ToLower(unames), "linux") == false {
			return false, nil
		}

		return true, nil
	},
}

var unixFamily = &PlatformResolver{
	Name:     FAMILY_UNIX,
	Familiy:  true,
	Children: []*PlatformResolver{bsdFamily, linuxFamily, solaris},
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		// in order to support linux container image detection, we cannot run
		// processes here, lets just read files to detect a system
		return true, nil
	},
}

var solaris = &PlatformResolver{
	Name:    "solaris",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		osrd := NewOSReleaseDetector(t)

		// check if we got vmkernel
		unames, err := osrd.unames()
		if err != nil {
			return false, err
		}

		if strings.Contains(strings.ToLower(unames), "sunos") == false {
			return false, nil
		}

		// try to read the architecture
		unamem, err := osrd.unamem()
		if err == nil {
			di.Arch = unamem
		}

		di.Name = "solaris"

		// NOTE: we have only one solaris system here, since we only get here is the familiy is sunos, we pass

		// try to read "/etc/release" for more details
		f, err := t.File("/etc/release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := ioutil.ReadAll(f)
		if err != nil {
			return false, nil
		}

		r, err := ParseSolarisRelease(string(c))
		if err == nil {
			di.Name = r.ID
			di.Title = r.Title
			di.Release = r.Release
		}

		return true, nil
	},
}

var esxi = &PlatformResolver{
	Name:    "esxi",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		log.Debug().Msg("check for esxi system")
		// at this point, we are already 99% its esxi
		cmd, err := t.RunCommand("vmware -v")
		if err != nil {
			log.Debug().Err(err).Msg("could not run command")
			return false, nil
		}
		vmware_info, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			log.Debug().Err(err).Msg("could not run command")
			return false, err
		}

		version, err := ParseEsxiRelease(string(vmware_info))
		if err != nil {
			log.Debug().Err(err).Msg("could not run command")
			return false, err
		}

		di.Release = version
		return true, nil
	},
}

var esxFamily = &PlatformResolver{
	Name:     "esx",
	Familiy:  true,
	Children: []*PlatformResolver{esxi},
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		osrd := NewOSReleaseDetector(t)

		// check if we got vmkernel
		unames, err := osrd.unames()
		if err != nil {
			return false, err
		}

		if strings.Contains(strings.ToLower(unames), "vmkernel") == false {
			return false, nil
		}

		di.Name = "esxi"

		// try to read the architecture
		unamem, err := osrd.unamem()
		if err == nil {
			di.Arch = unamem
		}

		return true, nil
	},
}

var windowsFamily = &PlatformResolver{
	Name:     FAMILY_WINDOWS,
	Familiy:  true,
	Children: []*PlatformResolver{windows},
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		return true, nil
	},
}

var unknownOperatingSystem = &PlatformResolver{
	Name:    "unknown-os",
	Familiy: false,
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		// if we reach here, we really do not know the system
		log.Debug().Msg("platform> we do not know the operating system, please contact support")
		return true, nil
	},
}

var operatingSystems = &PlatformResolver{
	Name:     "os",
	Familiy:  true,
	Children: []*PlatformResolver{unixFamily, windowsFamily, esxFamily, unknownOperatingSystem},
	Detect: func(p *PlatformResolver, di *PlatformInfo, t types.Transport) (bool, error) {
		return true, nil
	},
}
