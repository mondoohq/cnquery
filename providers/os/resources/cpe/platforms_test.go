// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cpe

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPlatformCPE(t *testing.T) {

	type testdata struct {
		platform    string
		version     string
		workstation bool
		cpe         string
	}

	tests := []testdata{
		{
			platform:    "windows",
			version:     "10000",
			workstation: true,
			cpe:         "cpe:2.3:o:microsoft:windows:10:*:*:*:*:*:*:*",
		},
		{
			platform:    "windows",
			version:     "20000",
			workstation: true,
			cpe:         "cpe:2.3:o:microsoft:windows:11:*:*:*:*:*:*:*",
		},
		{
			platform: "windows",
			version:  "14393",
			cpe:      "cpe:2.3:o:microsoft:windows_server_2016:-:*:*:*:*:*:*:*",
		},
		{
			platform: "windows",
			version:  "17763",
			cpe:      "cpe:2.3:o:microsoft:windows_server_2019:-:*:*:*:*:*:*:*",
		},
		{
			platform: "windows",
			version:  "20348",
			cpe:      "cpe:2.3:o:microsoft:windows_server_2022:-:*:*:*:*:*:*:*",
		},
		{
			platform: "debian",
			version:  "10.7",
			cpe:      "cpe:2.3:o:debian:debian_linux:10.7:*:*:*:*:*:*:*",
		},
		{
			platform: "macos",
			version:  "10.14",
			cpe:      "cpe:2.3:o:apple:mac_os_x:10.14.0:*:*:*:*:*:*:*",
		},
		{
			platform: "aix",
			version:  "7.2",
			cpe:      "cpe:2.3:o:ibm:aix:7.2:*:*:*:*:*:*:*",
		},
		{
			platform: "alpine",
			version:  "3.18.4",
			cpe:      "cpe:2.3:o:alpinelinux:alpine_linux:3.18.4:*:*:*:*:*:*:*",
		},
		{
			platform: "ubuntu",
			version:  "20.04",
			cpe:      "cpe:2.3:o:canonical:ubuntu_linux:20.04:*:*:*:lts:*:*:*",
		},
		{
			platform: "ubuntu",
			version:  "22.04",
			cpe:      "cpe:2.3:o:canonical:ubuntu_linux:22.04:*:*:*:lts:*:*:*",
		},
		{
			platform: "amazonlinux",
			version:  "2",
			cpe:      "cpe:2.3:o:amazon:linux_2:-:*:*:*:*:*:*:*",
		},
		{
			platform: "amazonlinux",
			version:  "2023",
			cpe:      "cpe:2.3:o:amazon:linux_2023:-:*:*:*:*:*:*:*",
		},
	}

	for _, test := range tests {
		cpe, ok := PlatformCPE(test.platform, test.version, test.workstation)
		assert.True(t, ok)
		assert.Equal(t, test.cpe, cpe, "platform: %s, version: %s", test.platform, test.version)
	}
}
