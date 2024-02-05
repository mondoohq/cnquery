// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

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

	sshParams, err := ParseBlocks(string(raw))
	if err != nil {
		t.Fatalf("cannot request file %v", err)
	}

	assert.NotNil(t, sshParams, "params are not nil")

	// check result for multiple host-keys
	assert.Equal(t, "/etc/ssh/ssh_host_rsa_key,/etc/ssh/ssh_host_ecdsa_key,/etc/ssh/ssh_host_ed25519_key", sshParams[0].Params["HostKey"])
	assert.Equal(t, "yes", sshParams[0].Params["X11Forwarding"])
	assert.Equal(t, "60", sshParams[0].Params["LoginGraceTime"])
}

func TestSSHParseCaseInsensitive(t *testing.T) {
	raw, err := os.ReadFile("./testdata/case_insensitive")
	require.NoError(t, err)

	sshParams, err := ParseBlocks(string(raw))
	if err != nil {
		t.Fatalf("cannot request file %v", err)
	}

	assert.NotNil(t, sshParams, "params are not nil")

	assert.Equal(t, "22", sshParams[0].Params["Port"])
	assert.Equal(t, "any", sshParams[0].Params["AddressFamily"])
	assert.Equal(t, "0.0.0.0", sshParams[0].Params["ListenAddress"])
}
