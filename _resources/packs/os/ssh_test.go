package os_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/resources/packs/os"
	"go.mondoo.com/cnquery/resources/packs/testutils"
)

func TestResource_SSHD(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), os.Registry)
	x.TestSimpleErrors(t, []testutils.SimpleTest{
		{
			"sshd.config('1').params['2'] == '3'",
			0, "failed to create resource 'sshd.config': could not load sshd configuration: 1",
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
}

func TestResource_MultiFileSSHD(t *testing.T) {
	x := testutils.InitTester(testutils.CustomMock("../testdata/fedora.toml"), os.Registry)

	t.Run("sshd params", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.params")
		require.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("sshd param only defined in referenced included file", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.ciphers")
		require.NotEmpty(t, res)
		assert.Contains(t, res[0].Data.Value, "aes256-gcm@openssh.com")
	})

	t.Run("sshd param only defined file with relative path", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.params")
		require.NotEmpty(t, res)
		data, ok := res[0].Data.Value.(map[string]interface{})
		require.True(t, ok, "failed to convert resulting values")
		require.Contains(t, data, "Port", "didn't find 'Port' param set")
		assert.Equal(t, data["Port"], "35", "expected value from file-by-reference not set")
	})

	t.Run("sshd param only defined in file inside of directory", func(t *testing.T) {
		res := x.TestQuery(t, "sshd.config.params")
		require.NotEmpty(t, res)
		data, ok := res[0].Data.Value.(map[string]interface{})
		require.True(t, ok, "failed to convert resulting values")
		require.Contains(t, data, "MaxSessions", "didn't find 'MaxSessions' param set")
		assert.Equal(t, data["MaxSessions"], "99", "expected value from file in directory not set")
	})
}
