// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cpe

import (
	"regexp"
	"strconv"
)

var platformCPES = []platformCPEEntry{
	// apple macos
	{
		Platform: "macos",
		Version:  "10.14",
		CPE:      "cpe:2.3:o:apple:mac_os_x:10.14.0:*:*:*:*:*:*:*",
	},
	{
		Platform: "macos",
		Version:  "10.15",
		CPE:      "cpe:2.3:o:apple:mac_os_x:10.15.0:*:*:*:*:*:*:*",
	},
	{
		Platform:     "macos",
		VersionRegex: regexp.MustCompile(`^11\.`),
		CPE:          "cpe:2.3:o:apple:mac_os_x:11.0.0:*:*:*:*:*:*:*",
	},
	{
		Platform:     "macos",
		VersionRegex: regexp.MustCompile(`^12\.`),
		CPE:          "cpe:2.3:o:apple:mac_os_x:12.0.0:*:*:*:*:*:*:*",
	},
	{
		Platform:     "macos",
		VersionRegex: regexp.MustCompile(`^13\.`),
		CPE:          "cpe:2.3:o:apple:mac_os_x:13.0.0:*:*:*:*:*:*:*",
	},
	{
		Platform:     "macos",
		VersionRegex: regexp.MustCompile(`^14\.`),
		CPE:          "cpe:2.3:o:apple:mac_os_x:14.0.0:*:*:*:*:*:*:*",
	},
	// amazon linux
	{
		Platform: "amazonlinux",
		Version:  "2",
		CPE:      "cpe:2.3:o:amazon:linux_2:-:*:*:*:*:*:*:*",
	},
	{
		Platform:     "amazonlinux",
		VersionRegex: regexp.MustCompile(`^(2017|2018)\.`),
		CPE:          "cpe:2.3:o:amazon:linux:-:*:*:*:*:*:*:*",
	},
	{
		Platform: "amazonlinux",
		Version:  "2023",
		CPE:      "cpe:2.3:o:amazon:linux_2023:-:*:*:*:*:*:*:*",
	},
	// centos
	{
		Platform:     "centos",
		VersionRegex: regexp.MustCompile(`^6\.`),
		CPE:          "cpe:2.3:o:centos:centos:6:*:*:*:*:*:*:*",
	},
	{
		Platform:     "centos",
		VersionRegex: regexp.MustCompile(`^7\.`),
		CPE:          "cpe:2.3:o:centos:centos:7:*:*:*:*:*:*:*",
	},
	{
		Platform:     "centos",
		VersionRegex: regexp.MustCompile(`^8\.`),
		CPE:          "cpe:2.3:o:centos:centos:8:*:*:*:*:*:*:*",
	},
	// debian
	{
		Platform:     "debian",
		VersionRegex: regexp.MustCompile(`^8\.`),
		CPE:          "cpe:2.3:o:debian:debian_linux:8.*:*:*:*:*:*:*:*",
	},
	{
		Platform:     "debian",
		VersionRegex: regexp.MustCompile(`^9\.`),
		CPE:          "cpe:2.3:o:debian:debian_linux:9.*:*:*:*:*:*:*:*",
	},
	{
		Platform:     "debian",
		VersionRegex: regexp.MustCompile(`^10\.`),
		CPE:          "cpe:2.3:o:debian:debian_linux:10:*:*:*:*:*:*:*",
	},
	{
		Platform:     "debian",
		VersionRegex: regexp.MustCompile(`^11\.`),
		CPE:          "cpe:2.3:o:debian:debian_linux:11:*:*:*:*:*:*:*",
	},
	// fedora
	{
		Platform: "fedora",
		Version:  "28",
		CPE:      "cpe:2.3:o:fedora:linux:28:*:*:*:*:*:*:*",
	},
	// oracle linux
	{
		Platform:     "oraclelinux",
		VersionRegex: regexp.MustCompile(`^6\.`),
		CPE:          "cpe:2.3:o:oracle:linux:6:*:*:*:*:*:*:*",
	},
	{
		Platform:     "oraclelinux",
		VersionRegex: regexp.MustCompile(`^7\.`),
		CPE:          "cpe:2.3:o:oracle:linux:7:*:*:*:*:*:*:*",
	},
	{
		Platform:     "oraclelinux",
		VersionRegex: regexp.MustCompile(`^8\.`),
		CPE:          "cpe:2.3:o:oracle:linux:8:*:*:*:*:*:*:*",
	},
	{
		Platform:     "oraclelinux",
		VersionRegex: regexp.MustCompile(`^9\.`),
		CPE:          "cpe:2.3:o:oracle:linux:9:*:*:*:*:*:*:*",
	},
	// redhat linux
	{
		Platform:     "redhat",
		VersionRegex: regexp.MustCompile(`^6\.`),
		CPE:          "cpe:2.3:o:redhat:redhat_enterprise_linux:6.*:*:*:en:*:*:*:*",
	},
	{
		Platform:     "redhat",
		VersionRegex: regexp.MustCompile(`^7\.`),
		CPE:          "cpe:2.3:o:redhat:redhat_enterprise_linux:7.*:*:*:en:*:*:*:*",
	},
	{
		Platform:     "redhat",
		VersionRegex: regexp.MustCompile(`^8\.`),
		CPE:          "cpe:2.3:o:redhat:redhat_enterprise_linux:8.*:*:*:en:*:*:*:*",
	},
	{
		Platform:     "redhat",
		VersionRegex: regexp.MustCompile(`^9\.`),
		CPE:          "cpe:2.3:o:redhat:redhat_enterprise_linux:9.*:*:*:en:*:*:*:*",
	},
	// rockylinux
	{
		Platform:     "rockylinux",
		VersionRegex: regexp.MustCompile(`^8\.`),
		CPE:          "cpe:2.3:o:rocky:rocky_linux:8.*:*:*:*:*:*:*:*",
	},
	{
		Platform:     "rockylinux",
		VersionRegex: regexp.MustCompile(`^9\.`),
		CPE:          "cpe:2.3:o:rocky:rocky_linux:9.*:*:*:*:*:*:*:*",
	},
	// alma linux
	{
		Platform:     "almalinux",
		VersionRegex: regexp.MustCompile(`^8\.`),
		CPE:          "cpe:2.3:o:almalinux:almalinux:8:*:*:*:*:*:*:*",
	},
	{
		Platform:     "almalinux",
		VersionRegex: regexp.MustCompile(`^9\.`),
		CPE:          "cpe:2.3:o:almalinux:almalinux:9:*:*:*:*:*:*:*",
	},
	// suse
	{
		Platform:     "sles",
		VersionRegex: regexp.MustCompile(`^12\.`),
		CPE:          "cpe:2.3:o:suse:suse_linux_enterprise_server:12*:*:*:*:*:*:*:*",
	},
	{
		Platform:     "sles",
		VersionRegex: regexp.MustCompile(`^15\.`),
		CPE:          "cpe:2.3:o:suse:suse_linux_enterprise_server:15*:*:*:*:*:*:*:*",
	},
	// ubuntu
	{
		Platform: "ubuntu",
		Version:  "16.04",
		CPE:      "cpe:2.3:o:canonical:ubuntu_linux:16.04:*:*:*:*:*:*:*",
	},
	{
		Platform: "ubuntu",
		Version:  "18.04",
		CPE:      "cpe:2.3:o:canonical:ubuntu_linux:18.04:*:*:*:*:*:*:*",
	},
	{
		Platform: "ubuntu",
		Version:  "20.04",
		CPE:      "cpe:2.3:o:canonical:ubuntu_linux:20.04:*:*:*:lts:*:*:*",
	},
	{
		Platform: "ubuntu",
		Version:  "22.04",
		CPE:      "cpe:2.3:o:canonical:ubuntu_linux:22.04:*:*:*:lts:*:*:*",
	},
	// windows
	{
		Platform: "windows",
		VersionFunc: func(v string, workstation bool) bool {
			version, err := strconv.Atoi(v)
			if err != nil {
				return false
			}
			return version >= 10000 && version < 20000 && workstation
		},
		CPE:         "cpe:2.3:o:microsoft:windows:10:*:*:*:*:*:*:*",
		Workstation: true,
	},
	{
		Platform: "windows",
		VersionFunc: func(v string, workstation bool) bool {
			version, err := strconv.Atoi(v)
			if err != nil {
				return false
			}
			return version >= 20000 && version < 30000 && workstation
		},
		CPE:         "cpe:2.3:o:microsoft:windows_11:-:*:*:*:*:*:x64:*",
		Workstation: true,
	},
	{
		Platform: "windows",
		Version:  "14393",
		CPE:      "cpe:2.3:o:microsoft:windows_server_2016:-:*:*:*:*:*:*:*",
	},
	{
		Platform: "windows",
		Version:  "17763",
		CPE:      "cpe:2.3:o:microsoft:windows_server_2019:-:*:*:*:*:*:*:*",
	},
	{
		Platform: "windows",
		Version:  "20348",
		CPE:      "cpe:2.3:o:microsoft:windows_server:2022:*:*:*:*:*:*:*",
	},
	// aix
	{
		Platform:     "aix",
		VersionRegex: regexp.MustCompile(`^7\.`),
		CPE:          "cpe:2.3:o:ibm:aix:7:*:*:*:*:*:*:*",
	},
}

func PlatformCPE(platform string, version string, workstation bool) (string, bool) {

	for i := range platformCPES {
		entry := platformCPES[i]
		if entry.Platform == platform {
			if entry.VersionFunc != nil {
				if entry.VersionFunc(version, workstation) {
					return entry.CPE, true
				}
			}
			if entry.VersionRegex != nil {
				if entry.VersionRegex.MatchString(version) {
					return entry.CPE, true
				}
			} else {
				if entry.Version == version {
					return entry.CPE, true
				}
			}
		}
	}

	return "", false
}

type platformCPEEntry struct {
	Platform     string
	VersionFunc  func(version string, workstation bool) bool
	VersionRegex *regexp.Regexp
	Version      string
	CPE          string
	Workstation  bool
}
