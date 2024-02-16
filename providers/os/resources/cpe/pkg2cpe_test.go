// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cpe

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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
