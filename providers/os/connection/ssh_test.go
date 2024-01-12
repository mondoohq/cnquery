// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
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
