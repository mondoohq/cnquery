package users_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/users"
	mock "go.mondoo.io/mondoo/motor/mock/toml"
	"go.mondoo.io/mondoo/motor/types"
)

func TestParseEtcPasswd(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "users_linux.toml"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := mock.File("/etc/passwd")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	r, err := f.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	assert.Nil(t, err)

	m, err := users.ParseEtcPasswd(r)
	assert.Nil(t, err)
	assert.Equal(t, 13, len(m), "detected the right amount of services")

	assert.Equal(t, "root", m[0].Username, "detected user name")
	assert.Equal(t, int64(0), m[0].Uid, "detected uid")
	assert.Equal(t, int64(0), m[0].Gid, "detected gid")
	assert.Equal(t, "root", m[0].Description, "user description")
	assert.Equal(t, "/root", m[0].Home, "detected user home")
	assert.Equal(t, "/bin/bash", m[0].Shell, "detected user shell")
}

func TestParseDsclListResult(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "users_osx.toml"})
	if err != nil {
		t.Fatal(err)
	}

	// check user shells
	c, err := mock.RunCommand("dscl . -list /Users UserShell")
	if err != nil {
		t.Fatal(err)
	}

	m, err := users.ParseDsclListResult(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 100, len(m), "detected the right amount of users")
	assert.Equal(t, "/usr/bin/false", m["_analyticsd"], "detected uid name")

	// check uid
	c, err = mock.RunCommand("dscl . -list /Users UniqueID")
	if err != nil {
		t.Fatal(err)
	}

	m, err = users.ParseDsclListResult(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 100, len(m), "detected the right amount of users")
	assert.Equal(t, "263", m["_analyticsd"], "detected uid name")
}
