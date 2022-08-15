package ssh_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/ssh"
)

func TestSSHProviderError(t *testing.T) {
	_, err := ssh.New(&providers.Config{Backend: providers.ProviderType_LOCAL_OS, Host: "example.local"})
	assert.Equal(t, "provider type does not match", err.Error())
}

func TestSSHAuthError(t *testing.T) {
	_, err := ssh.New(&providers.Config{Backend: providers.ProviderType_SSH, Host: "example.local"})

	assert.True(t,
		// local testing if ssh agent is available
		err.Error() == "dial tcp: lookup example.local: no such host" ||
			// local testing without ssh agent
			err.Error() == "no authentication method defined")
}
