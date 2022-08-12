package ssh_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/ssh"
)

func TestSSHBackendError(t *testing.T) {
	_, err := ssh.New(&providers.TransportConfig{Backend: providers.ProviderType_LOCAL_OS, Host: "example.local"})
	assert.Equal(t, "only ssh backend for ssh transport supported", err.Error())
}

func TestSSHAuthError(t *testing.T) {
	_, err := ssh.New(&providers.TransportConfig{Backend: providers.ProviderType_SSH, Host: "example.local"})

	assert.True(t,
		// local testing if ssh agent is available
		err.Error() == "dial tcp: lookup example.local: no such host" ||
			// local testing without ssh agent
			err.Error() == "no authentication method defined")
}
