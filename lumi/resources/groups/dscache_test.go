package groups_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/groups"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestParseDscacheutilResult(t *testing.T) {
	mock, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: "./testdata/osx.toml"})
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
