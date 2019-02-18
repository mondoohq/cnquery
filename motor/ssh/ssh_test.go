package ssh_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/ssh"
	"go.mondoo.io/mondoo/motor/types"
)

func TestSSHBackendError(t *testing.T) {
	_, err := ssh.New(&types.Endpoint{Backend: "ssh2", Host: "example.local"})
	assert.Equal(t, "only ssh backend for ssh transport supported", err.Error())
}

func TestSSHAuthError(t *testing.T) {
	_, err := ssh.New(&types.Endpoint{Backend: "ssh", Host: "example.local"})
	assert.Equal(t, "no authentication method defined", err.Error())
}

func TestSSHPort(t *testing.T) {

	endpoint := &types.Endpoint{Backend: "ssh", Host: "example.local", Password: "example"}
	err := ssh.VerifyConfig(endpoint)
	assert.Nil(t, err)

	endpoint = ssh.DefaultConfig(endpoint)

	// if no port is provided, it needs to be 22
	assert.Equal(t, 22, endpoint.Port)
}
