// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package groups_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v9/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v9/providers/os/resources/groups"
)

func TestParseLinuxEtcGroups(t *testing.T) {
	mock, err := mock.New("./testdata/debian.toml", nil)
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.FileSystem().Open("/etc/group")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)
	defer f.Close()

	m, err := groups.ParseEtcGroup(f)
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
	mock, err := mock.New("./testdata/freebsd12.toml", nil)
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.FileSystem().Open("/etc/group")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)
	defer f.Close()

	m, err := groups.ParseEtcGroup(f)
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
