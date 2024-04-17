// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package groups_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v11/providers/os/resources/groups"
)

func TestParseDscacheutilResult(t *testing.T) {
	mock, err := mock.New(0, "./testdata/osx.toml", &inventory.Asset{})
	if err != nil {
		t.Fatal(err)
	}

	c, err := mock.RunCommand("dscacheutil -q group")
	if err != nil {
		t.Fatal(err)
	}

	m, err := groups.ParseDscacheutilResult(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(m), "detected the right amount of groups")

	grp := findGroup(m, "395")
	assert.Equal(t, int64(395), grp.Gid, "detected group id")
	assert.Equal(t, "com.apple.access_ftp", grp.Name, "detected group name")
	assert.Equal(t, []string{}, grp.Members, "detected group members")

	grp = findGroup(m, "216")
	assert.Equal(t, int64(216), grp.Gid, "detected group id")
	assert.Equal(t, "_postgres", grp.Name, "detected group name")
	assert.Equal(t, []string{"_devicemgr", "_calendar", "_teamsserver", "_xserverdocs"}, grp.Members, "detected group members")
}
