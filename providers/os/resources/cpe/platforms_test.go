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
			platform: "debian",
			version:  "10.7",
			cpe:      "cpe:2.3:o:debian:debian_linux:10:*:*:*:*:*:*:*",
		},
		{
			platform: "macos",
			version:  "10.14",
			cpe:      "cpe:2.3:o:apple:mac_os_x:10.14.0:*:*:*:*:*:*:*",
		},
		{
			platform: "aix",
			version:  "7.2",
			cpe:      "cpe:2.3:o:ibm:aix:7:*:*:*:*:*:*:*",
		},
	}

	for _, test := range tests {
		cpe, ok := PlatformCPE(test.platform, test.version, test.workstation)
		assert.True(t, ok)
		assert.Equal(t, test.cpe, cpe, "platform: %s, version: %s", test.platform, test.version)
	}
}
