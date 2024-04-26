// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
)

func TestResource_SSHD(t *testing.T) {
	x.TestSimpleErrors(t, []testutils.SimpleTest{
		{
			Code:        "sshd.config('nopath').params['2'] == '3'",
			ResultIndex: 0,
			Expectation: "file '/etc/ssh/nopath' not found",
		},
	})

	t.Run("sshd file path", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.file.path")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("sshd params", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.params")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("sshd file error propagation", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config('nope').params")
		assert.Error(t, res[0].Data.Error)
	})

	t.Run("specific sshd param", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.params[\"UsePAM\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "yes", res[0].Data.Value)
	})

	t.Run("parse ciphers", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.ciphers")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{"chacha20-poly1305@openssh.com", "aes256-gcm@openssh.com", "aes128-gcm@openssh.com", "aes256-ctr", "aes192-ctr", "aes128-ctr"}, res[0].Data.Value)
	})

	t.Run("parse macs", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.macs")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{"hmac-sha2-512-etm@openssh.com", "hmac-sha2-256-etm@openssh.com", "umac-128-etm@openssh.com", "hmac-sha2-512", "hmac-sha2-256"}, res[0].Data.Value)
	})

	t.Run("parse kexs", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.kexs")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{"curve25519-sha256@libssh.org", "diffie-hellman-group-exchange-sha256"}, res[0].Data.Value)
	})

	t.Run("parse hostKeys", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.hostkeys")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{"/etc/ssh/ssh_host_rsa_key", "/etc/ssh/ssh_host_ecdsa_key", "/etc/ssh/ssh_host_ed25519_key"}, res[0].Data.Value)
	})

	t.Run("parse permitRootLogin", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.permitRootLogin")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{"no"}, res[0].Data.Value)
	})

	t.Run("parse blocks", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.blocks.map(criteria)")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []any{"", "Group sftp-users", "User myservice"}, res[0].Data.Value)

		res = x.TestQuery(t, "sshd.config.blocks.map(params.AllowTcpForwarding)")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []any{"no", "yes", nil}, res[0].Data.Value)
	})
	t.Run("expose block match criteria in params.Match", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.params.Match")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Group sftp-users,User myservice", res[0].Data.Value)
	})
}
