package yum

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestParseYumRepoEntry(t *testing.T) {
	data := `
Repo-id      : base/7/x86_64
Repo-name    : CentOS-7 - Base
Repo-status  : enabled
Repo-revision: 1587512243
Repo-updated : Tue Apr 21 23:37:50 2020
Repo-pkgs    : 10070
Repo-size    : 8.9 G
Repo-mirrors : http://mirrorlist.centos.org/?release=7&arch=x86_64&repo=os&infra=container
Repo-baseurl : http://mirror.imt-systems.com/centos/7.8.2003/os/x86_64/ (9 more)
Repo-expire  : 21600 second(s) (last: Tue Jun 16 07:13:59 2020)
	Filter     : read-only:present
Repo-filename: /etc/yum.repos.d/CentOS-Base.repo

Repo-id      : c7-media
Repo-name    : CentOS-7 - Media
Repo-status  : disabled
Repo-baseurl : file:///media/CentOS/, file:///media/cdrom/, file:///media/cdrecorder/
Repo-expire  : 21600 second(s) (last: Unknown)
  Filter     : read-only:present
Repo-filename: /etc/yum.repos.d/CentOS-Media.repo
	`
	repos, err := ParseRepos(strings.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, 2, len(repos))

	repo := repos[0]
	assert.Equal(t, "base/7/x86_64", repo.Id)
	assert.Equal(t, "CentOS-7 - Base", repo.Name)
	assert.Equal(t, "enabled", repo.Status)
	assert.Equal(t, "1587512243", repo.Revision)
	assert.Equal(t, "Tue Apr 21 23:37:50 2020", repo.Updated)
	assert.Equal(t, "10070", repo.Pkgs)
	assert.Equal(t, "8.9 G", repo.Size)
	assert.Equal(t, "http://mirrorlist.centos.org/?release=7&arch=x86_64&repo=os&infra=container", repo.Mirrors)
	assert.Equal(t, []string{"http://mirror.imt-systems.com/centos/7.8.2003/os/x86_64/"}, repo.Baseurl)
	assert.Equal(t, "21600 second(s) (last: Tue Jun 16 07:13:59 2020)", repo.Expire)
	assert.Equal(t, "read-only:present", repo.Filter)
	assert.Equal(t, "/etc/yum.repos.d/CentOS-Base.repo", repo.Filename)

	repo = repos[1]
	assert.Equal(t, "c7-media", repo.Id)
	assert.Equal(t, "CentOS-7 - Media", repo.Name)
	assert.Equal(t, "disabled", repo.Status)
	assert.Equal(t, []string{"file:///media/CentOS/", "file:///media/cdrom/", "file:///media/cdrecorder/"}, repo.Baseurl)
}

func TestYumRepoRhel7(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/yum_rhel7.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	cmd, err := m.Transport.RunCommand(RhelYumRepoListCommand)
	require.NoError(t, err)
	repos, err := ParseRepos(cmd.Stdout)
	require.NoError(t, err)
	assert.Equal(t, 15, len(repos))

	cmd, err = m.Transport.RunCommand(Rhel6VarsCommand)
	require.NoError(t, err)
	vars, err := ParseVariables(cmd.Stdout)
	require.NoError(t, err)
	assert.Equal(t, "7Server", vars["releasever"])
}

func TestYumRepoRhel8(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/yum_rhel8.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	cmd, err := m.Transport.RunCommand(RhelYumRepoListCommand)
	require.NoError(t, err)
	repos, err := ParseRepos(cmd.Stdout)
	require.NoError(t, err)
	assert.Equal(t, 17, len(repos))

	cmd, err = m.Transport.RunCommand(Rhel8VarsCommand)
	require.NoError(t, err)
	vars, err := ParseVariables(cmd.Stdout)
	require.NoError(t, err)
	assert.Equal(t, "8", vars["releasever"])
}
