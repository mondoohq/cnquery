// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
)

func TestWinOSUpdatesParser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/updates_win.toml", &inventory.Asset{
		Platform: &inventory.Platform{Name: "windows"},
	})
	if err != nil {
		t.Fatal(err)
	}

	cmd := powershell.Encode(WINDOWS_QUERY_WSUS_AVAILABLE)
	c, err := mock.RunCommand(cmd)
	if err != nil {
		t.Fatal(err)
	}

	m, err := ParseWindowsUpdates(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(m), "detected the right amount of packages")

	pkg, err := findKb(m, "890830")
	require.NoError(t, err)
	assert.Equal(t, "890830", pkg.Name, "update id detected")
	assert.Equal(t, "Windows Malicious Software Removal Tool x64 - March 2020 (KB890830)", pkg.Description, "update title detected")

	pkg, err = findKb(m, "4538461")
	require.NoError(t, err)
	assert.Equal(t, "4538461", pkg.Name, "update id detected")
	assert.Equal(t, "2020-03 Cumulative Update for Windows Server 2019 (1809) for x64-based Systems (KB4538461)", pkg.Description, "update title detected")

	// check empty return
	m, err = ParseWindowsUpdates(strings.NewReader(""))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(m), "detected the right amount of packages")
}

func findKb(pkgs []OperatingSystemUpdate, name string) (OperatingSystemUpdate, error) {
	for i := range pkgs {
		if pkgs[i].Name == name {
			return pkgs[i], nil
		}
	}

	return OperatingSystemUpdate{}, errors.New("not found")
}
