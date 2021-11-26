package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_SSHD(t *testing.T) {
	t.Run("sshd params", func(t *testing.T) {
		res := testQuery(t, "sshd.config.params")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("sshd file error propagation", func(t *testing.T) {
		res := testQuery(t, "sshd.config('nope').params")
		assert.Error(t, res[0].Data.Error)
	})

	t.Run("specific sshs param", func(t *testing.T) {
		res := testQuery(t, "sshd.config.params[\"UsePAM\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "yes", res[0].Data.Value)
	})

	t.Run("parse ciphers", func(t *testing.T) {
		res := testQuery(t, "sshd.config.ciphers")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{"chacha20-poly1305@openssh.com", "aes256-gcm@openssh.com", "aes128-gcm@openssh.com", "aes256-ctr", "aes192-ctr", "aes128-ctr"}, res[0].Data.Value)
	})

	t.Run("parse macs", func(t *testing.T) {
		res := testQuery(t, "sshd.config.macs")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{"hmac-sha2-512-etm@openssh.com", "hmac-sha2-256-etm@openssh.com", "umac-128-etm@openssh.com", "hmac-sha2-512", "hmac-sha2-256"}, res[0].Data.Value)
	})

	t.Run("parse kexs", func(t *testing.T) {
		res := testQuery(t, "sshd.config.kexs")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{"curve25519-sha256@libssh.org", "diffie-hellman-group-exchange-sha256"}, res[0].Data.Value)
	})

	t.Run("parse hostKeys", func(t *testing.T) {
		res := testQuery(t, "sshd.config.hostkeys")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{"/etc/ssh/ssh_host_rsa_key", "/etc/ssh/ssh_host_ecdsa_key", "/etc/ssh/ssh_host_ed25519_key"}, res[0].Data.Value)
	})
}
