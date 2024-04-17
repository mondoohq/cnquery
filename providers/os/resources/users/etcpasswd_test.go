// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package users_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v11/providers/os/resources/users"
)

func TestParseLinuxEtcPasswd(t *testing.T) {
	mock, err := mock.New(0, "./testdata/debian.toml", &inventory.Asset{})
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
	mock, err := mock.New(0, "./testdata/freebsd12.toml", &inventory.Asset{})
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
