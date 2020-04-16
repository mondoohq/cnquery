package groups_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/groups"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestParseDscacheutilResult(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/osx.toml"})
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

	assert.Equal(t, int64(395), m[0].Gid, "detected group id")
	assert.Equal(t, "com.apple.access_ftp", m[0].Name, "detected group name")
	assert.Equal(t, []string{}, m[0].Members, "detected group members")

	assert.Equal(t, int64(216), m[2].Gid, "detected group id")
	assert.Equal(t, "_postgres", m[2].Name, "detected group name")
	assert.Equal(t, []string{"_devicemgr", "_calendar", "_teamsserver", "_xserverdocs"}, m[2].Members, "detected group members")
}
