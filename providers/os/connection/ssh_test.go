// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

func TestSSHDefaultSettings(t *testing.T) {
	conn := &SshConnection{
		conf: &inventory.Config{
			Sudo: &inventory.Sudo{
				Active: true,
			},
		},
	}
	conn.setDefaultSettings()
	assert.Equal(t, int32(22), conn.conf.Port)
	assert.Equal(t, "sudo", conn.conf.Sudo.Executable)
}

func TestSSHProviderError(t *testing.T) {
	_, err := NewSshConnection(0, &inventory.Config{Type: shared.Type_Local.String(), Host: "example.local"}, &inventory.Asset{})
	assert.Equal(t, "provider type does not match", err.Error())
}

func TestSSHAuthError(t *testing.T) {
	_, err := NewSshConnection(0, &inventory.Config{Type: shared.Type_SSH.String(), Host: "example.local"}, &inventory.Asset{})
	assert.True(t,
		// local testing if ssh agent is available
		err.Error() == "dial tcp: lookup example.local: no such host" ||
			// local testing without ssh agent
			err.Error() == "no authentication method defined")
}
