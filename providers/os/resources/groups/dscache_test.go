// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package groups_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/providers/os/resources/groups"
)

func TestParseDscacheutilResult(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/osx.toml"))
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

func TestParseDscacheutilResult_InvalidGid(t *testing.T) {
	// A group whose gid cannot be parsed must be dropped rather than recorded
	// with a bogus (unparseable) ID and gid 0.
	input := `name: brokengroup
password: *
gid: notanumber
users: alice

name: validgroup
password: *
gid: 501
users: bob carol
`
	res, err := groups.ParseDscacheutilResult(strings.NewReader(input))
	require.NoError(t, err)

	require.Len(t, res, 1)
	assert.Equal(t, "validgroup", res[0].Name)
	assert.Equal(t, int64(501), res[0].Gid)
	assert.Equal(t, "501", res[0].ID)
	assert.Nil(t, findGroup(res, "notanumber"))
}
