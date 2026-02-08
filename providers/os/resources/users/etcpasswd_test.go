// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package users_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/providers/os/resources/users"
)

func TestParseLinuxEtcPasswd(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/debian.toml"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.FileSystem().Open("/etc/passwd")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)
	defer f.Close()

	m, err := users.ParseEtcPasswd(f)
	assert.Nil(t, err)
	assert.Equal(t, 13, len(m), "detected the right amount of services")

	assert.Equal(t, "root", m[0].Name, "detected user name")
	assert.Equal(t, int64(0), m[0].Uid, "detected uid")
	assert.Equal(t, int64(0), m[0].Gid, "detected gid")
	assert.Equal(t, "root", m[0].Description, "user description")
	assert.Equal(t, "/root", m[0].Home, "detected user home")
	assert.Equal(t, "/bin/bash", m[0].Shell, "detected user shell")
}

func TestParseFreebsdLinuxEtcPasswd(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/freebsd12.toml"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.FileSystem().Open("/etc/passwd")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)
	defer f.Close()

	m, err := users.ParseEtcPasswd(f)
	assert.Nil(t, err)
	assert.Equal(t, 28, len(m), "detected the right amount of services")

	assert.Equal(t, "root", m[0].Name, "detected user name")
	assert.Equal(t, int64(0), m[0].Uid, "detected uid")
	assert.Equal(t, int64(0), m[0].Gid, "detected gid")
	assert.Equal(t, "Charlie &", m[0].Description, "user description")
	assert.Equal(t, "/root", m[0].Home, "detected user home")
	assert.Equal(t, "/bin/csh", m[0].Shell, "detected user shell")
}

func TestParseLinuxGetentPasswd(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"os", "unix", "linux", "redhat"},
		},
	}, mock.WithPath("./testdata/oraclelinux_getent_passwd.toml"))
	require.NoError(t, err)
	m, err := users.ResolveManager(conn)
	require.Nil(t, err)

	list, err := m.List()
	require.Nil(t, err)
	assert.Equal(t, 20, len(list), "detected the right amount of users")

	assert.Equal(t, "root", list[0].Name, "detected user name")
	assert.Equal(t, int64(0), list[0].Uid, "detected uid")
	assert.Equal(t, int64(0), list[0].Gid, "detected gid")
	assert.Equal(t, "root", list[0].Description, "user description")
	assert.Equal(t, "/root", list[0].Home, "detected user home")
	assert.Equal(t, "/bin/bash", list[0].Shell, "detected user shell")
}
