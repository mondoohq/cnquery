package sshd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHParser(t *testing.T) {
	raw, err := os.ReadFile("./testdata/sshd_config")
	require.NoError(t, err)

	sshParams, err := Params(string(raw))
	if err != nil {
		t.Fatalf("cannot request file %v", err)
	}

	assert.NotNil(t, sshParams, "params are not nil")

	// check result for multiple host-keys
	assert.Equal(t, "/etc/ssh/ssh_host_rsa_key,/etc/ssh/ssh_host_ecdsa_key,/etc/ssh/ssh_host_ed25519_key", sshParams["HostKey"])
	assert.Equal(t, "yes", sshParams["X11Forwarding"])
	assert.Equal(t, "60", sshParams["LoginGraceTime"])
}

func TestSSHParseCaseInsensitive(t *testing.T) {
	raw, err := os.ReadFile("./testdata/case_insensitive")
	require.NoError(t, err)

	sshParams, err := Params(string(raw))
	if err != nil {
		t.Fatalf("cannot request file %v", err)
	}

	assert.NotNil(t, sshParams, "params are not nil")

	assert.Equal(t, "22", sshParams["Port"])
	assert.Equal(t, "any", sshParams["AddressFamily"])
	assert.Equal(t, "0.0.0.0", sshParams["ListenAddress"])
}
