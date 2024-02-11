// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
)

// SUSE OS updates
func TestZypperPatchParser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/updates_zypper.toml", &inventory.Asset{
		Platform: &inventory.Platform{Name: "suse"},
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("zypper --xmlout list-updates -t patch")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := ParseZypperPatches(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(m), "detected the right amount of packages")

	assert.Equal(t, "openSUSE-2018-397", m[0].Name, "update name detected")
	assert.Equal(t, "moderate", m[0].Severity, "severity version detected")
}
