// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

// UBR - Update Build Revision
func TestParseWinRegistryCurrentVersion(t *testing.T) {

	t.Run("parse windows version", func(t *testing.T) {
		data := `{
			"CurrentBuild":  "17763",
			"UBR":  720,
			"EditionID": "ServerDatacenterEval",
			"ReleaseId": "1809"
		}`

		m, err := ParseWinRegistryCurrentVersion(strings.NewReader(data))
		assert.Nil(t, err)

		assert.Equal(t, "17763", m.CurrentBuild, "buildnumber should be parsed properly")
		assert.Equal(t, 720, m.UBR, "ubr should be parsed properly")
	})

	t.Run("parse windows version with architecture", func(t *testing.T) {
		data := `{
			"CurrentBuild":  "26100",
			"UBR":  2033,
			"InstallationType":  "Client",
			"EditionID":  "Enterprise",
			"ProductName":  "Windows 10 Enterprise",
			"DisplayVersion":  "24H2",
			"Architecture":  "ARM64",
			"ProductType":  "WinNT"
		}`
		m, err := ParseWinRegistryCurrentVersion(strings.NewReader(data))
		assert.Nil(t, err)

		assert.Equal(t, "26100", m.CurrentBuild, "buildnumber should be parsed properly")
		assert.Equal(t, 2033, m.UBR, "ubr should be parsed properly")
		assert.Equal(t, "ARM64", m.Architecture, "architecture should be parsed properly")
	})
}
