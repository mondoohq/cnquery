package users_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/users"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestParseDsclListResult(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/osx.toml")
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
	assert.Equal(t, 8, len(m), "detected the right amount of users")
	assert.Equal(t, "/usr/bin/false", m["_www"], "detected uid name")

	// check uid
	c, err = mock.RunCommand("dscl . -list /Users UniqueID")
	if err != nil {
		t.Fatal(err)
	}

	m, err = users.ParseDsclListResult(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 8, len(m), "detected the right amount of users")
	assert.Equal(t, "70", m["_www"], "detected uid name")

	// check user home
	c, err = mock.RunCommand("dscl . -list /Users NFSHomeDirectory")
	if err != nil {
		t.Fatal(err)
	}

	m, err = users.ParseDsclListResult(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 7, len(m), "detected the right amount of users")
	assert.Equal(t, "/Library/WebServer", m["_www"], "detected uid name")
	assert.Equal(t, "/var/root /private/var/root", m["root"], "detected root name")
}
