package ssh_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/ssh"
)

func TestSSHBackendError(t *testing.T) {
	_, err := ssh.New(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_LOCAL_OS, Host: "example.local"})
	assert.Equal(t, "only ssh backend for ssh transport supported", err.Error())
}

func TestSSHAuthError(t *testing.T) {
	_, err := ssh.New(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_SSH, Host: "example.local"})

	assert.True(t,
		// local testing if ssh agent is available
		err.Error() == "dial tcp: lookup example.local: no such host" ||
			// local testing without ssh agent
			err.Error() == "no authentication method defined")
}

func TestSSHPort(t *testing.T) {
	endpoint := &transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_SSH, Host: "example.local"}
	err := ssh.VerifyConfig(endpoint)
	assert.Nil(t, err)

	endpoint = ssh.DefaultConfig(endpoint)

	// if no port is provided, it needs to be 22
	assert.Equal(t, "22", endpoint.Port)
}
