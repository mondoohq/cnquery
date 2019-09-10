package platform

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

type detect func(p *PlatformResolver, di *Info) (bool, error)

type PlatformResolver struct {
	Name     string
	Familiy  bool
	Children []*PlatformResolver
	Detect   detect
}

// often used family names
const (
	FAMILY_UNIX    = "unix"
	FAMILY_DARWIN  = "darwin"
	FAMILY_LINUX   = "linux"
	FAMILY_WINDOWS = "windows"
)

func (p *PlatformResolver) Resolve() (bool, *Info) {
	// prepare detect info object
	di := &Info{}
	di.Family = make([]string, 0)

	// start recursive platform resolution
	ok, pi := p.resolvePlatform(di)
	log.Debug().Str("platform", pi.Name).Strs("family", pi.Family).Msg("platform> detected os")
	return ok, pi
}

// Resolve tries to find recursively all
// platforms until a leaf (operating systems) detect
// mechanism is returning true
func (p *PlatformResolver) resolvePlatform(di *Info) (bool, *Info) {
	detected, err := p.Detect(p, di)
	if err != nil {
		return false, di
	}

	// if detection is true but we have a family
	if detected == true && p.Familiy == true {
		// we are a familiy and we may have childs to try
		for _, c := range p.Children {
			resolved, detected := c.resolvePlatform(di)
			if resolved {
				// add family hieracy
				detected.Family = append(di.Family, p.Name)
				return resolved, detected
			}
		}

		// we reached this point, we know it is the platfrom but we could not
		// identify the system
		// TODO: add generic platform instance
		// TODO: should we return an error?
	}

	// return if the detect is true and we have a leaf
	if detected && p.Familiy == false {
		return true, di
	}

	// could not find it
	return false, di
}

func (d *Detector) buildPlatformTree() (*PlatformResolver, error) {

	// Operating Systems
	macOS := &PlatformResolver{
		Name:    "macos",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			// when we reach here, we know it is darwin
			// check xml /System/Library/CoreServices/SystemVersion.plist

			f, err := d.Transport.File("/System/Library/CoreServices/SystemVersion.plist")
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
	otherDarwin := &PlatformResolver{
		Name:    "darwin",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			return true, nil
		},
	}

	alpine := &PlatformResolver{
		Name:    "alpine",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			if di.Name == "alpine" {
				return true, nil
			}

			f, err := d.Transport.File("/etc/alpine-release")
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

	arch := &PlatformResolver{
		Name:    "arch",
		Familiy: true,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			if di.Name == "arch" {
				return true, nil
			}

			// arch has no version number, use kernel instead
			// TODO: be aware that this does not work for containers
			uname, err := d.unamer()
			if err == nil {
				di.Release = uname
			}

			return false, nil
		},
	}

	manjaro := &PlatformResolver{
		Name:    "manjaro",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			if di.Name == "manjaro" {
				return true, nil
			}

			return false, nil
		},
	}

	debian := &PlatformResolver{
		Name:    "debian",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {

			f, err := d.Transport.File("/etc/debian_version")
			if err != nil {
				return false, nil
			}
			defer f.Close()

			c, err := ioutil.ReadAll(f)
			if err != nil || len(c) == 0 {
				return false, nil
			}

			osr, err := d.osrelease()
			if err != nil {
				return false, nil
			}

			if osr["ID"] != "debian" {
				return false, nil
			}

			di.Title = osr["NAME"]
			di.Release = strings.TrimSpace(string(c))

			unamem, err := d.unamem()
			if err == nil {
				di.Arch = unamem
			}

			return true, nil
		},
	}

	ubuntu := &PlatformResolver{
		Name:    "ubuntu",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			if di.Name == "ubuntu" {
				return true, nil
			}
			return false, nil
		},
	}

	raspbian := &PlatformResolver{
		Name:    "raspbian",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			if di.Name == "raspbian" {
				return true, nil
			}
			return false, nil
		},
	}

	rhel := &PlatformResolver{
		Name:    "redhat",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
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

	centos := &PlatformResolver{
		Name:    "centos",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			// works for centos 5+
			if strings.Contains(di.Title, "CentOS") || di.Name == "centos" {
				di.Name = "centos"
				return true, nil
			}

			// CentOS 5 does not have /etc/centos-release
			// check if we have /etc/centos-release file
			f, err := d.Transport.File("/etc/centos-release")
			if err != nil {
				return false, nil
			}
			defer f.Close()

			c, err := ioutil.ReadAll(f)
			if err != nil || len(c) == 0 {
				return false, nil
			}

			if len(di.Name) == 0 {
				di.Name = "centos"
			}

			return true, nil
		},
	}

	fedora := &PlatformResolver{
		Name:    "fedora",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			if strings.Contains(di.Title, "Fedora") || di.Name == "fedora" {
				di.Name = "fedora"
				return true, nil
			}

			// check if we have /etc/fedora-release file
			f, err := d.Transport.File("/etc/fedora-release")
			if err != nil {
				return false, nil
			}
			defer f.Close()

			c, err := ioutil.ReadAll(f)
			if err != nil || len(c) == 0 {
				return false, nil
			}

			if len(di.Name) == 0 {
				di.Name = "centos"
			}

			return true, nil
		},
	}

	oracle := &PlatformResolver{
		Name:    "oracle",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			// works for oracle 7+
			if di.Name == "ol" {
				return true, nil
			}

			// check if we have /etc/centos-release file
			f, err := d.Transport.File("/etc/oracle-release")
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

	scientific := &PlatformResolver{
		Name:    "scientific",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
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

	amazonlinux := &PlatformResolver{
		Name:    "amzn",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			if di.Name == "amzn" {
				return true, nil
			}
			return false, nil
		},
	}
	opensuse := &PlatformResolver{
		Name:    "opensuse",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			if di.Name == "opensuse" || di.Name == "opensuse-leap" {
				return true, nil
			}

			return false, nil
		},
	}
	sles := &PlatformResolver{
		Name:    "sles",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			if di.Name == "sles" {
				return true, nil
			}
			return false, nil
		},
	}
	gentoo := &PlatformResolver{
		Name:    "gentoo",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			f, err := d.Transport.File("/etc/gentoo-release")
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

	busybox := &PlatformResolver{
		Name:    "busybox",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {

			command := "ls --help 2>&1 | head -1"
			cmd, err := d.Transport.RunCommand(command)
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

	// fallback linux detection, since we do not know the system, the family detection may not be correct
	defaultLinux := &PlatformResolver{
		Name:    "generic-linux",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			// if we reach here, we know that we detected linux already
			log.Debug().Msg("platform> we do not know the linux system, but we do our best in guessing")
			return true, nil
		},
	}

	windows := &PlatformResolver{
		Name:    "windows",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			// wmic is available since Windows Server 2008/Vista
			command := "wmic os get * /format:csv"
			cmd, err := d.Transport.RunCommand(command)
			if err != nil {
				return false, nil
			}

			data, err := ParseWinWmicOS(cmd.Stdout)
			if err != nil {
				return false, nil
			}

			di.Name = "windows"
			di.Title = data.Caption

			// major.minor.build.ubr
			di.Release = data.Version

			// FIXME: we need to ask wmic cpu get architecture
			di.Arch = data.OSArchitecture

			// optional: try to get the ubr number (win 10 + 2019)
			pscommand := "Get-ItemProperty -Path 'HKLM:\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion' -Name CurrentBuild, UBR | ConvertTo-Json"
			cmd, err = d.Transport.RunCommand(fmt.Sprintf("powershell -c \"%s\"", pscommand))
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
	darwinFamily := &PlatformResolver{
		Name:     FAMILY_DARWIN,
		Familiy:  true,
		Children: []*PlatformResolver{macOS, otherDarwin},
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			unames, err := d.unames()
			if err != nil {
				return false, err
			}

			if strings.Contains(strings.ToLower(unames), "darwin") == false {
				return false, nil
			}

			// from here we know it is a darwin system
			unamem, err := d.unamem()
			if err == nil {
				di.Arch = unamem
			}

			// read information from /usr/bin/sw_vers
			dsv, err := d.darwin_swversion()
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

	bsdFamily := &PlatformResolver{
		Name:     "bsd",
		Familiy:  true,
		Children: []*PlatformResolver{darwinFamily},
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			unames, err := d.unames()
			if err != nil {
				return false, err
			}

			if len(unames) > 0 {
				return true, nil
			}
			return false, nil
		},
	}

	redhatFamily := &PlatformResolver{
		Name:     "redhat",
		Familiy:  true,
		Children: []*PlatformResolver{rhel, centos, fedora, oracle, scientific},
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			f, err := d.Transport.File("/etc/redhat-release")
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

	debianFamily := &PlatformResolver{
		Name:     "debian",
		Familiy:  true,
		Children: []*PlatformResolver{debian, ubuntu, raspbian},
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			return true, nil
		},
	}

	suseFamily := &PlatformResolver{
		Name:     "suse",
		Familiy:  true,
		Children: []*PlatformResolver{opensuse, sles},
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			return true, nil
		},
	}

	archFamily := &PlatformResolver{
		Name:     "arch",
		Familiy:  true,
		Children: []*PlatformResolver{arch, manjaro},
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			f, err := d.Transport.File("/etc/arch-release")
			if err != nil {
				return false, nil
			}
			defer f.Close()

			c, err := ioutil.ReadAll(f)
			if err != nil {
				return false, nil
			}
			if len(c) == 0 {
				return false, nil
			}
			return true, nil
		},
	}

	linuxFamily := &PlatformResolver{
		Name:     FAMILY_LINUX,
		Familiy:  true,
		Children: []*PlatformResolver{archFamily, redhatFamily, debianFamily, suseFamily, amazonlinux, alpine, gentoo, busybox, defaultLinux},
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			detected := false

			lsb, err := d.lsbconfig()
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

			osr, err := d.osrelease()
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
			f, err := d.Transport.File("/etc/redhat-release")
			if f != nil {
				f.Close()
			}

			if err == nil {
				detected = true
			}

			// try to read the architecture, we cannot assume this works if we use the tar bakcend where we
			// just load the filesystem, therefore we do not fail here
			unamem, err := d.unamem()
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
			unames, err := d.unames()
			if err != nil {
				return false, err
			}

			if strings.Contains(strings.ToLower(unames), "linux") == false {
				return false, nil
			}

			return true, nil
		},
	}

	unixFamily := &PlatformResolver{
		Name:     FAMILY_UNIX,
		Familiy:  true,
		Children: []*PlatformResolver{bsdFamily, linuxFamily},
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			// in order to support linux container image detection, we cannot run
			// processes here, lets just read files to detect a system
			return true, nil
		},
	}

	windowsFamily := &PlatformResolver{
		Name:     FAMILY_WINDOWS,
		Familiy:  true,
		Children: []*PlatformResolver{windows},
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			return true, nil
		},
	}

	unknownOperatingSystem := &PlatformResolver{
		Name:    "unknown-os",
		Familiy: false,
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			// if we reach here, we really do not know the system
			log.Debug().Msg("platform> we do not know the operating system, please contact support")
			return true, nil
		},
	}

	operatingSystem := &PlatformResolver{
		Name:     "os",
		Familiy:  true,
		Children: []*PlatformResolver{windowsFamily, unixFamily, unknownOperatingSystem},
		Detect: func(p *PlatformResolver, di *Info) (bool, error) {
			return true, nil
		},
	}

	return operatingSystem, nil
}
