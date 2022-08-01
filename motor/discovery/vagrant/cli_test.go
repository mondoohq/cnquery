package vagrant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/discovery/vagrant"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestVagrantSshConfigParsing(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/vagrant.toml")
	require.NoError(t, err)

	cmd, err := mock.RunCommand("vagrant ssh-config debian10")
	require.NoError(t, err)

	config, err := vagrant.ParseVagrantSshConfig(cmd.Stdout)
	require.NoError(t, err)

	assert.Equal(t, 1, len(config))
	assert.Equal(t, "debian10", config["debian10"].Host)
	assert.Equal(t, ".vagrant/machines/debian10/virtualbox/private_key", config["debian10"].IdentityFile)
	assert.Equal(t, "vagrant", config["debian10"].User)
	assert.Equal(t, int(2222), config["debian10"].Port)
}

func TestVagrantStatusParsing(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/vagrant.toml")
	require.NoError(t, err)

	cmd, err := mock.RunCommand("vagrant status")
	require.NoError(t, err)

	vms, err := vagrant.ParseVagrantStatus(cmd.Stdout)
	require.NoError(t, err)

	assert.Equal(t, 9, len(vms))
	assert.Equal(t, true, vms["debian10"])
	assert.Equal(t, false, vms["ubuntu1604"])
}
