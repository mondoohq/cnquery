// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

const (
	LabelDistroID = "distro-id"
)

// Operating Systems
var macOS = &PlatformResolver{
	Name:     "macos",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		// when we reach here, we know it is darwin
		// check xml /System/Library/CoreServices/SystemVersion.plist
		f, err := conn.FileSystem().Open("/System/Library/CoreServices/SystemVersion.plist")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := io.ReadAll(f)
		if err != nil || len(c) == 0 {
			return false, nil
		}

		sv, err := ParseMacOSSystemVersion(string(c))
		if err != nil || len(c) == 0 {
			return false, nil
		}

		pf.Name = "macos"
		pf.Title = sv["ProductName"]
		pf.Version = sv["ProductVersion"]
		pf.Build = sv["ProductBuildVersion"]

		return true, nil
	},
}

// is part of the darwin platform and fallback for non-known darwin systems
var otherDarwin = &PlatformResolver{
	Name:     "darwin",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		return true, nil
	},
}

var alpine = &PlatformResolver{
	Name:     "alpine",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		// check if we are on edge
		osrd := NewOSReleaseDetector(conn)
		osr, err := osrd.osrelease()
		if err != nil {
			return false, nil
		}

		if osr["PRETTY_NAME"] == "Alpine Linux edge" {
			pf.Name = "alpine"
			pf.Version = "edge"
			pf.Build = osr["VERSION_ID"]
		}

		// if we are on alpine, the release was detected properly from parent check
		if pf.Name == "alpine" {
			return true, nil
		}

		f, err := conn.FileSystem().Open("/etc/alpine-release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := io.ReadAll(f)
		if err != nil || len(c) == 0 {
			return false, nil
		}

		pf.Name = "alpine"
		return true, nil
	},
}

var arch = &PlatformResolver{
	Name:     "arch",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "arch" {
			return true, nil
		}
		return false, nil
	},
}

var manjaro = &PlatformResolver{
	Name:     "manjaro",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "manjaro" {
			return true, nil
		}
		return false, nil
	},
}

var debian = &PlatformResolver{
	Name:     "debian",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		osrd := NewOSReleaseDetector(conn)

		f, err := conn.FileSystem().Open("/etc/debian_version")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := io.ReadAll(f)
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

		pf.Version = strings.TrimSpace(string(c))

		unamem, err := osrd.unamem()
		if err == nil {
			pf.Arch = unamem
		}

		return true, nil
	},
}

var ubuntu = &PlatformResolver{
	Name:     "ubuntu",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "ubuntu" {
			return true, nil
		}
		return false, nil
	},
}

var raspbian = &PlatformResolver{
	Name:     "raspbian",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "raspbian" {
			return true, nil
		}
		return false, nil
	},
}

var kali = &PlatformResolver{
	Name:     "kali",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "kali" {
			return true, nil
		}
		return false, nil
	},
}

var linuxmint = &PlatformResolver{
	Name:     "linuxmint",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "linuxmint" {
			return true, nil
		}
		return false, nil
	},
}

var popos = &PlatformResolver{
	Name:     "pop",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "pop" {
			return true, nil
		}
		return false, nil
	},
}

// rhel PlatformResolver only detects redhat and no derivatives
var rhel = &PlatformResolver{
	Name:     "redhat",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		// etc redhat release was parsed by the family already,
		// we reuse that information here
		// e.g. Red Hat Linux, Red Hat Enterprise Linux Server
		if strings.Contains(pf.Title, "Red Hat") || pf.Name == "redhat" {
			pf.Name = "redhat"
			return true, nil
		}

		// fallback to /etc/redhat-release file
		f, err := conn.FileSystem().Open("/etc/redhat-release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := io.ReadAll(f)
		if err != nil || len(c) == 0 {
			return false, nil
		}

		if strings.Contains(string(c), "Red Hat") {
			pf.Name = "redhat"
			return true, nil
		}

		return false, nil
	},
}

var eurolinux = &PlatformResolver{
	Name:     "eurolinux",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "eurolinux" {
			return true, nil
		}
		return false, nil
	},
}

// The centos platform resolver finds CentOS and CentOS-like platforms like alma and rocky
var centos = &PlatformResolver{
	Name:     "centos",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		// works for centos 5+
		if strings.Contains(pf.Title, "CentOS") || pf.Name == "centos" {
			pf.Name = "centos"
			return true, nil
		}

		// adapt the name for rocky to align it with amazonlinux, almalinux etc.
		if pf.Name == "rocky" {
			pf.Name = "rockylinux"
		}

		// newer alma linux do not have /etc/centos-release, check for alma linux
		afs := &afero.Afero{Fs: conn.FileSystem()}
		if pf.Name == "almalinux" {
			if ok, err := afs.Exists("/etc/almalinux-release"); err == nil && ok {
				return true, nil
			}
		}

		// newer rockylinux do not have /etc/centos-release
		if pf.Name == "rockylinux" {
			if ok, err := afs.Exists("/etc/rocky-release"); err == nil && ok {
				return true, nil
			}
		}

		// NOTE: CentOS 5 does not have /etc/centos-release
		// fallback to /etc/centos-release file
		if ok, err := afs.Exists("/etc/centos-release"); err != nil || !ok {
			return false, nil
		}

		if len(pf.Name) == 0 {
			pf.Name = "centos"
		}

		return true, nil
	},
}

var fedora = &PlatformResolver{
	Name:     "fedora",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if strings.Contains(pf.Title, "Fedora") || pf.Name == "fedora" {
			pf.Name = "fedora"
			return true, nil
		}

		// fallback to /etc/fedora-release file
		f, err := conn.FileSystem().Open("/etc/fedora-release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := io.ReadAll(f)
		if err != nil || len(c) == 0 {
			return false, nil
		}

		if len(pf.Name) == 0 {
			pf.Name = "fedora"
		}

		return true, nil
	},
}

var oracle = &PlatformResolver{
	Name:     "oracle",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		// works for oracle 7+
		if pf.Name == "ol" {
			pf.Name = "oraclelinux"
			return true, nil
		}

		// check if we have /etc/centos-release file
		f, err := conn.FileSystem().Open("/etc/oracle-release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := io.ReadAll(f)
		if err != nil || len(c) == 0 {
			return false, nil
		}

		if len(pf.Name) == 0 {
			pf.Name = "oraclelinux"
		}

		return true, nil
	},
}

var scientific = &PlatformResolver{
	Name:     "scientific",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		// works for oracle 7+
		if pf.Name == "scientific" {
			return true, nil
		}

		// we only get here if this is a rhel distribution
		if strings.Contains(pf.Title, "Scientific Linux") {
			pf.Name = "scientific"
			return true, nil
		}

		return false, nil
	},
}

var amazonlinux = &PlatformResolver{
	Name:     "amazonlinux",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "amzn" {
			pf.Name = "amazonlinux"
			return true, nil
		}
		return false, nil
	},
}

var windriver = &PlatformResolver{
	Name:     "wrlinux",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "wrlinux" {
			return true, nil
		}
		return false, nil
	},
}

var opensuse = &PlatformResolver{
	Name:     "opensuse",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "opensuse" || pf.Name == "opensuse-leap" || pf.Name == "opensuse-tumbleweed" {
			return true, nil
		}

		return false, nil
	},
}

var sles = &PlatformResolver{
	Name:     "sles",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "sles" {
			return true, nil
		}
		return false, nil
	},
}

var suseMicroOs = &PlatformResolver{
	Name:     "suse-microos",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "suse-microos" {
			return true, nil
		}
		return false, nil
	},
}

var gentoo = &PlatformResolver{
	Name:     "gentoo",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		f, err := conn.FileSystem().Open("/etc/gentoo-release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := io.ReadAll(f)
		if err != nil || len(c) == 0 {
			log.Debug().Err(err)
			return false, nil
		}

		content := strings.TrimSpace(string(c))
		name, release, err := ParseRhelVersion(content)
		if err == nil {
			// only set title if not already properly detected by lsb or os-release
			if len(pf.Title) == 0 {
				pf.Title = name
			}
			if len(pf.Version) == 0 {
				pf.Version = release
			}
		}

		return false, nil
	},
}

var ubios = &PlatformResolver{
	Name:     "ubios",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "ubios" {
			return true, nil
		}
		return false, nil
	},
}

var busybox = &PlatformResolver{
	Name:     "busybox",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		busyboxExists, err := afero.Exists(conn.FileSystem(), "/bin/busybox")
		if !busyboxExists || err != nil {
			return false, nil
		}

		// we need to read this file because all others show up as zero size
		// This fille seems to be the "original"
		// all others are hardlinks
		f, err := conn.FileSystem().Open("/bin/[")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		content, err := io.ReadAll(f)
		if err != nil {
			return false, err
		}

		// strings are \0 terminated
		rodataByteStrings := bytes.Split(content, []byte("\x00"))
		if rodataByteStrings == nil {
			return false, nil
		}

		releaseRegex := regexp.MustCompile(`^(.+)\s(v[\d\.]+)\s*\((.*)\).*$`)
		for _, rodataByteString := range rodataByteStrings {
			rodataString := string(rodataByteString)
			m := releaseRegex.FindStringSubmatch(rodataString)
			if len(m) >= 2 {
				title := m[1]
				release := m[2]

				if strings.ToLower(title) == "busybox" {
					pf.Name = "busybox"
					pf.Title = title
					pf.Version = release
					return true, nil
				}
			}
		}

		return false, nil
	},
}

var photon = &PlatformResolver{
	Name:     "photon",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if pf.Name == "photon" {
			return true, nil
		}
		return false, nil
	},
}

var openwrt = &PlatformResolver{
	Name:     "openwrt",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		// No clue why they are not using either lsb-release or os-release
		f, err := conn.FileSystem().Open("/etc/openwrt_release")
		if err != nil {
			return false, err
		}
		defer f.Close()

		content, err := io.ReadAll(f)
		if err != nil {
			return false, err
		}

		lsb, err := ParseLsbRelease(string(content))
		if err == nil {
			if len(lsb["DISTRIB_ID"]) > 0 {
				pf.Name = strings.ToLower(lsb["DISTRIB_ID"])
				pf.Title = lsb["DISTRIB_ID"]
			}
			if len(lsb["DISTRIB_RELEASE"]) > 0 {
				pf.Version = lsb["DISTRIB_RELEASE"]
			}

			return true, nil
		}

		return false, nil
	},
}

var (
	plcnextVersion      = regexp.MustCompile(`(?m)^Arpversion:\s+(.*)$`)
	plcnextBuildVersion = regexp.MustCompile(`(?m)^GIT Commit Hash:\s+(.*)$`)
)

var plcnext = &PlatformResolver{
	Name:     "plcnext",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		// No clue why they are not using either lsb-release or os-release
		f, err := conn.FileSystem().Open("/etc/plcnext/arpversion")
		if err != nil {
			return false, err
		}
		defer f.Close()

		content, err := io.ReadAll(f)
		if err != nil {
			return false, err
		}

		m := plcnextVersion.FindStringSubmatch(string(content))
		if len(m) >= 2 {
			pf.Name = "plcnext"
			pf.Title = "PLCnext"
			pf.Version = m[1]

			bm := plcnextBuildVersion.FindStringSubmatch(string(content))
			if len(bm) >= 2 {
				pf.Build = bm[1]
			}

			return true, err
		}

		return false, nil
	},
}

// fallback linux detection, since we do not know the system, the family detection may not be correct
var defaultLinux = &PlatformResolver{
	Name:     "generic-linux",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		// if we reach here, we know that we detected linux already
		log.Debug().Msg("platform> we do not know the linux system, but we do our best in guessing")
		return true, nil
	},
}

var netbsd = &PlatformResolver{
	Name:     "netbsd",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if strings.Contains(strings.ToLower(pf.Name), "netbsd") == false {
			return false, nil
		}

		osrd := NewOSReleaseDetector(conn)
		release, err := osrd.unamer()
		if err == nil {
			pf.Version = release
		}

		return true, nil
	},
}

var freebsd = &PlatformResolver{
	Name:     "freebsd",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if strings.Contains(strings.ToLower(pf.Name), "freebsd") == false {
			return false, nil
		}

		osrd := NewOSReleaseDetector(conn)
		release, err := osrd.unamer()
		if err == nil {
			pf.Version = release
		}

		return true, nil
	},
}

var openbsd = &PlatformResolver{
	Name:     "openbsd",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if strings.Contains(strings.ToLower(pf.Name), "openbsd") == false {
			return false, nil
		}

		osrd := NewOSReleaseDetector(conn)
		release, err := osrd.unamer()
		if err == nil {
			pf.Version = release
		}

		return true, nil
	},
}

var dragonflybsd = &PlatformResolver{
	Name:     "dragonflybsd",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if strings.Contains(strings.ToLower(pf.Name), "dragonfly") == false {
			return false, nil
		}

		pf.Name = "dragonflybsd"
		osrd := NewOSReleaseDetector(conn)
		release, err := osrd.unamer()
		if err == nil {
			pf.Version = release
		}

		return true, nil
	},
}

var windows = &PlatformResolver{
	Name:     "windows",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if conn.Capabilities().Has(shared.Capability_RunCommand) {
			return runtimeWindowsDetector(pf, conn)
		}

		if conn.Capabilities().Has(shared.Capability_FileSearch) {
			return staticWindowsDetector(pf, conn)
		}
		return false, nil
	},
}

var slugRe = regexp.MustCompile("[^a-z0-9]+")

func slugifyDarwin(s string) string {
	s = strings.ToLower(s)
	s = slugRe.ReplaceAllString(s, "_")
	return strings.Trim(s, "_")
}

// Families
var darwinFamily = &PlatformResolver{
	Name:     inventory.FAMILY_DARWIN,
	IsFamily: true,
	Children: []*PlatformResolver{macOS, otherDarwin},
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		if strings.Contains(strings.ToLower(pf.Name), "darwin") == false {
			return false, nil
		}
		// from here we know it is a darwin system

		// read information from /usr/bin/sw_vers
		osrd := NewOSReleaseDetector(conn)
		dsv, err := osrd.darwin_swversion()
		// ignore dsv config if we got an error
		if err == nil {
			if len(dsv["ProductName"]) > 0 {
				// name needs to be slugged
				pf.Name = slugifyDarwin(dsv["ProductName"])
				if pf.Name == "mac_os_x" {
					pf.Name = "macos"
				}
				pf.Title = dsv["ProductName"]
			}
			if len(dsv["ProductVersion"]) > 0 {
				pf.Version = dsv["ProductVersion"]
			}
		} else {
			// TODO: we know its darwin, but without swversion support
			log.Error().Err(err)
		}

		return true, nil
	},
}

var bsdFamily = &PlatformResolver{
	Name:     inventory.FAMILY_BSD,
	IsFamily: true,
	Children: []*PlatformResolver{darwinFamily, netbsd, freebsd, openbsd, dragonflybsd},
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		osrd := NewOSReleaseDetector(conn)
		unames, err := osrd.unames()
		if err != nil {
			return false, err
		}

		unamem, err := osrd.unamem()
		if err == nil {
			pf.Arch = unamem
		}

		if len(unames) > 0 {
			pf.Name = strings.ToLower(unames)
			pf.Title = unames
			return true, nil
		}
		return false, nil
	},
}

var redhatFamily = &PlatformResolver{
	Name:     "redhat",
	IsFamily: true,
	// NOTE: oracle pretends to be redhat with /etc/redhat-release and Red Hat Linux, therefore we
	// want to check that platform before redhat
	Children: []*PlatformResolver{oracle, rhel, centos, fedora, scientific, eurolinux},
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		f, err := conn.FileSystem().Open("/etc/redhat-release")
		if err != nil {
			log.Debug().Err(err)
			return false, nil
		}
		defer f.Close()

		c, err := io.ReadAll(f)
		if err != nil || len(c) == 0 {
			log.Debug().Err(err)
			return false, nil
		}

		content := strings.TrimSpace(string(c))
		title, release, err := ParseRhelVersion(content)
		if err == nil {
			log.Debug().Str("title", title).Str("release", release).Msg("detected rhelish platform")

			// only set title if not already properly detected by lsb or os-release
			if len(pf.Title) == 0 {
				pf.Title = title
			}

			// always override the version from the release file, since it is
			// more accurate
			if len(release) > 0 {
				pf.Version = release
			}

			return true, nil
		}

		return false, nil
	},
}

var debianFamily = &PlatformResolver{
	Name:     "debian",
	IsFamily: true,
	Children: []*PlatformResolver{debian, ubuntu, raspbian, kali, linuxmint, popos},
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		return true, nil
	},
}

var suseFamily = &PlatformResolver{
	Name:     "suse",
	IsFamily: true,
	Children: []*PlatformResolver{opensuse, sles, suseMicroOs},
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		return true, nil
	},
}

var archFamily = &PlatformResolver{
	Name:     "arch",
	IsFamily: true,
	Children: []*PlatformResolver{arch, manjaro},
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		// if the file exists, we are on arch or one of its derivatives
		f, err := conn.FileSystem().Open("/etc/arch-release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := io.ReadAll(f)
		if err != nil {
			return false, nil
		}

		// on arch containers, /etc/os-release may not be present
		if len(pf.Name) == 0 && strings.Contains(strings.ToLower(string(c)), "manjaro") {
			pf.Name = "manjaro"
			pf.Title = strings.TrimSpace(string(c))
			return true, nil
		}

		if len(pf.Name) == 0 {
			// fallback to arch
			pf.Name = "arch"
			pf.Title = "Arch Linux"
		}
		return true, nil
	},
}

var linuxFamily = &PlatformResolver{
	Name:     inventory.FAMILY_LINUX,
	IsFamily: true,
	Children: []*PlatformResolver{archFamily, redhatFamily, debianFamily, suseFamily, amazonlinux, alpine, gentoo, busybox, photon, windriver, openwrt, ubios, plcnext, defaultLinux},
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		detected := false
		osrd := NewOSReleaseDetector(conn)

		pf.Name = ""
		pf.Title = ""
		if pf.Labels == nil {
			pf.Labels = map[string]string{}
		}

		lsb, err := osrd.lsbconfig()
		// ignore lsb config if we got an error
		if err == nil {
			if len(lsb["DISTRIB_ID"]) > 0 {
				pf.Name = strings.ToLower(lsb["DISTRIB_ID"])
			}
			if len(lsb["DISTRIB_DESCRIPTION"]) > 0 {
				pf.Title = lsb["DISTRIB_DESCRIPTION"]
			} else if len(lsb["DISTRIB_ID"]) > 0 {
				pf.Title = lsb["DISTRIB_ID"]
			}
			if len(lsb["DISTRIB_RELEASE"]) > 0 {
				pf.Version = lsb["DISTRIB_RELEASE"]
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
				pf.Name = osr["ID"]
				pf.Labels[LabelDistroID] = osr["ID"]
			}
			if len(osr["PRETTY_NAME"]) > 0 {
				pf.Title = osr["PRETTY_NAME"]
			}
			if len(osr["VERSION_ID"]) > 0 {
				pf.Version = osr["VERSION_ID"]
			}

			if len(osr["BUILD_ID"]) > 0 {
				pf.Build = osr["BUILD_ID"]
			}

			detected = true
		}

		// Centos 6 does not include /etc/os-release or /etc/lsb-release, therefore any static analysis
		// will not be able to detect the system, since the following unamem and unames mechanism is not
		// available there. Instead the system can be identified by the availability of /etc/redhat-release
		// If /etc/redhat-release is available, we know its a linux system.
		f, err := conn.FileSystem().Open("/etc/redhat-release")
		if f != nil {
			f.Close()
		}

		if err == nil {
			detected = true
		}

		// BusyBox images do not contain /etc/os-release or /etc/lsb-release, therefore any static analysis
		// will not be able to detect the system, since the following unamem and unames mechanism is not
		// available there. Instead the system can be identified by the availability of /bin/busybox
		// If /bin/busybox is available, we know its a linux system.
		f, err = conn.FileSystem().Open("/bin/busybox")
		if f != nil {
			f.Close()
		}

		if err == nil {
			detected = true
		}

		// try to read the architecture, we cannot assume this works if we use the tar backend where we
		// just load the filesystem, therefore we do not fail here
		unamem, err := osrd.unamem()
		if err == nil {
			pf.Arch = unamem
		}

		// abort if os-release or lsb config was available, we don't need uname -s then
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
	Name:     inventory.FAMILY_UNIX,
	IsFamily: true,
	Children: []*PlatformResolver{bsdFamily, linuxFamily, solaris, aix},
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		// in order to support linux container image detection, we cannot run
		// processes here, lets just read files to detect a system
		return true, nil
	},
}

var solaris = &PlatformResolver{
	Name:     "solaris",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		osrd := NewOSReleaseDetector(conn)

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
			pf.Arch = unamem
		}

		pf.Name = "solaris"

		// NOTE: we have only one solaris system here, since we only get here is the family is sunos, we pass

		// try to read "/etc/release" for more details
		f, err := conn.FileSystem().Open("/etc/release")
		if err != nil {
			return false, nil
		}
		defer f.Close()

		c, err := io.ReadAll(f)
		if err != nil {
			return false, nil
		}

		release, err := ParseSolarisRelease(string(c))
		if err == nil {
			pf.Name = release.ID
			pf.Title = release.Title
			pf.Version = release.Release
		}

		return true, nil
	},
}

var aixUnameParser = regexp.MustCompile(`(\d+)\s+(\d+)\s+(.*)`)

var aix = &PlatformResolver{
	Name:     "aix",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		osrd := NewOSReleaseDetector(conn)

		unames, err := osrd.unames()
		if err != nil {
			return false, err
		}

		if strings.Contains(strings.ToLower(unames), "aix") == false {
			return false, nil
		}

		pf.Name = "aix"
		pf.Title = "AIX"

		// try to read the architecture and version
		unamervp, err := osrd.command("uname -rvp")
		if err == nil {
			m := aixUnameParser.FindStringSubmatch(unamervp)
			if len(m) == 4 {
				pf.Version = m[2] + "." + m[1]
				pf.Version = pf.Version
				pf.Arch = m[3]
			}
		}

		return true, nil
	},
}

var esxi = &PlatformResolver{
	Name:     "esxi",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		log.Debug().Msg("check for esxi system")
		// at this point, we are already 99% its esxi
		cmd, err := conn.RunCommand("vmware -v")
		if err != nil {
			log.Debug().Err(err).Msg("could not run command")
			return false, nil
		}
		vmware_info, err := io.ReadAll(cmd.Stdout)
		if err != nil {
			log.Debug().Err(err).Msg("could not run command")
			return false, err
		}

		version, err := ParseEsxiRelease(string(vmware_info))
		if err != nil {
			log.Debug().Err(err).Msg("could not run command")
			return false, err
		}

		pf.Version = version
		return true, nil
	},
}

var esxFamily = &PlatformResolver{
	Name:     "esx",
	IsFamily: true,
	Children: []*PlatformResolver{esxi},
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		osrd := NewOSReleaseDetector(conn)

		// check if we got vmkernel
		unames, err := osrd.unames()
		if err != nil {
			return false, err
		}

		if strings.Contains(strings.ToLower(unames), "vmkernel") == false {
			return false, nil
		}

		pf.Name = "esxi"

		// try to read the architecture
		unamem, err := osrd.unamem()
		if err == nil {
			pf.Arch = unamem
		}

		return true, nil
	},
}

var WindowsFamily = &PlatformResolver{
	Name:     inventory.FAMILY_WINDOWS,
	IsFamily: true,
	Children: []*PlatformResolver{windows},
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		return true, nil
	},
}

var unknownOperatingSystem = &PlatformResolver{
	Name:     "unknown-os",
	IsFamily: false,
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		// if we reach here, we really do not know the system
		log.Debug().Msg("platform> we do not know the operating system, please contact support")
		return true, nil
	},
}

var OperatingSystems = &PlatformResolver{
	Name:     "os",
	IsFamily: true,
	Children: []*PlatformResolver{unixFamily, WindowsFamily, esxFamily, unknownOperatingSystem},
	Detect: func(r *PlatformResolver, pf *inventory.Platform, conn shared.Connection) (bool, error) {
		return true, nil
	},
}
