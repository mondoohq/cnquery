// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cpe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPkg2Gen(t *testing.T) {
	tests := []struct {
		vendor   string
		name     string
		version  string
		expected string
	}{
		{"tar", "tar", "1.34+dfsg-1", "cpe:2.3:a:tar:tar:1.34\\+dfsg-1:*:*:*:*:*:*:*"},
		{"@coreui/vue", "@coreui/vue", "2.1.2", "cpe:2.3:a:\\@coreui\\/vue:\\@coreui\\/vue:2.1.2:*:*:*:*:*:*:*"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cpe, err := NewPackage2Cpe(test.vendor, test.name, test.version, "", "")
			require.NoError(t, err)
			assert.Equal(t, test.expected, cpe)
		})
	}
}

func TestPkg2GenMutations(t *testing.T) {
	tests := []struct {
		vendor   string
		name     string
		version  string
		arch     string
		expected []string
	}{
		{
			"nextgen_corporation",
			"mirth_connect_4.4.0.b2948",
			"4.4.0.b2948",
			"",
			[]string{
				"cpe:2.3:a:nextgen_corporation:mirth_connect_4.4.0.b2948:4.4.0.b2948:*:*:*:*:*:*:*",
				"cpe:2.3:a:nextgen:mirth_connect:4.4.0.b2948:*:*:*:*:*:*:*",
				"cpe:2.3:a:nextgen:mirth_connect:4.4.0:*:*:*:*:*:*:*",
			},
		},
		{
			"mirthconnect",
			"mirthconnect",
			"0:4.4.0.b2948-1",
			"i386",
			[]string{
				"cpe:2.3:a:nextgen:mirth_connect:4.4.0.b2948-1:*:*:*:*:*:*:*",
				"cpe:2.3:a:nextgen:mirth_connect:4.4.0:*:*:*:*:*:*:*",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cpes, err := NewPackage2Cpe(test.vendor, test.name, test.version, "", test.arch)
			require.NoError(t, err)
			for _, cpe := range test.expected {
				assert.Contains(t, cpes, cpe)
			}
		})
	}
}
