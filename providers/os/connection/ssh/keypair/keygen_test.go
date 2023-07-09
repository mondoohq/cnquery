package keypair

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateKeys(t *testing.T) {
	t.Run("generate rsa SSH keys", func(t *testing.T) {
		k, err := NewRSAKeys(DefaultRsaBits, nil)
		require.NoError(t, err, "error creating SSH key pair")
		require.True(t, len(k.PublicKey) > 0)
		require.True(t, len(k.PrivateKey) > 0)
	})

	t.Run("generate rsa SSH keys", func(t *testing.T) {
		k, err := NewRSAKeys(DefaultRsaBits, []byte("passphrase"))
		require.NoError(t, err, "error creating SSH key pair")
		require.True(t, len(k.PublicKey) > 0)
		require.True(t, len(k.PrivateKey) > 0)
	})

	t.Run("generate ed25519 SSH keys", func(t *testing.T) {
		k, err := NewEd25519Keys()
		require.NoError(t, err, "error creating SSH key pair")
		require.True(t, len(k.PublicKey) > 0)
		require.True(t, len(k.PrivateKey) > 0)
	})
}
