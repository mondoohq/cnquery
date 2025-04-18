// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package groups

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
)

func TestParseLinuxEtcGroups(t *testing.T) {
	mock, err := mock.New(0, "./testdata/debian.toml", &inventory.Asset{})
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.FileSystem().Open("/etc/group")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)
	defer f.Close()

	m, err := ParseEtcGroup(f)
	assert.Nil(t, err)
	assert.Equal(t, 23, len(m), "detected the right amount of services")

	assert.Equal(t, "root", m[0].Name, "detected user name")
	assert.Equal(t, "0", m[0].ID, "detected id")
	assert.Equal(t, int64(0), m[0].Gid, "detected gid")
	assert.Equal(t, "", m[22].Sid, "detected sid")
	assert.Equal(t, []string{}, m[0].Members, "user description")

	assert.Equal(t, "vagrant", m[22].Name, "detected user name")
	assert.Equal(t, "1000", m[22].ID, "detected id")
	assert.Equal(t, int64(1000), m[22].Gid, "detected gid")
	assert.Equal(t, "", m[22].Sid, "detected sid")
	assert.Equal(t, []string{"vagrant"}, m[22].Members, "user description")
}

func TestParseFreebsd12EtcGroups(t *testing.T) {
	mock, err := mock.New(0, "./testdata/freebsd12.toml", &inventory.Asset{})
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.FileSystem().Open("/etc/group")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)
	defer f.Close()

	m, err := ParseEtcGroup(f)
	assert.Nil(t, err)
	assert.Equal(t, 36, len(m), "detected the right amount of services")

	assert.Equal(t, "wheel", m[0].Name, "detected user name")
	assert.Equal(t, "0", m[0].ID, "detected id")
	assert.Equal(t, int64(0), m[0].Gid, "detected gid")
	assert.Equal(t, "", m[0].Sid, "detected sid")
	assert.Equal(t, []string{"root", "vagrant"}, m[0].Members, "user description")

	assert.Equal(t, "vagrant", m[35].Name, "detected user name")
	assert.Equal(t, "1001", m[35].ID, "detected id")
	assert.Equal(t, int64(1001), m[35].Gid, "detected gid")
	assert.Equal(t, "", m[35].Sid, "detected sid")
	assert.Equal(t, []string{}, m[35].Members, "user description")
}

func TestUnixGroupManager(t *testing.T) {
	mock, err := mock.New(0, "./testdata/debian.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"unix"},
		},
	})
	require.NoError(t, err)
	gm := &UnixGroupManager{
		conn: mock,
	}

	groups, err := gm.List()
	require.NoError(t, err)

	var vagrantGroup *Group
	for _, g := range groups {
		if g.Name == "vagrant" {
			vagrantGroup = g
			break
		}
	}

	require.NotNil(t, vagrantGroup)
	assert.Equal(t, "vagrant", vagrantGroup.Name)
	assert.Equal(t, int64(1000), vagrantGroup.Gid)
	assert.Equal(t, "1000", vagrantGroup.ID)
	assert.Equal(t, []string{"vagrant"}, vagrantGroup.Members)
}
