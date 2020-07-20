package ssh_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/ssh"
)

func TestSSHBackendError(t *testing.T) {
	_, err := ssh.New(&transports.Endpoint{Backend: "ssh2", Host: "example.local"})
	assert.Equal(t, "only ssh backend for ssh transport supported", err.Error())
}

func TestSSHAuthError(t *testing.T) {
	_, err := ssh.New(&transports.Endpoint{Backend: "ssh", Host: "example.local"})

	assert.True(t,
		// local testing if ssh agent is available
		err.Error() == "dial tcp: lookup example.local: no such host" ||
			// local testing without ssh agent
			err.Error() == "no authentication method defined")
}

func TestSSHPort(t *testing.T) {

	endpoint := &transports.Endpoint{Backend: "ssh", Host: "example.local", Password: "example"}
	err := ssh.VerifyConfig(endpoint)
	assert.Nil(t, err)

	endpoint = ssh.DefaultConfig(endpoint)

	// if no port is provided, it needs to be 22
	assert.Equal(t, "22", endpoint.Port)
}
