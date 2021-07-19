package multivault

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/vault"
	"go.mondoo.io/mondoo/motor/vault/inmemory"
)

func TestMultiVault(t *testing.T) {
	secret1 := &vault.Secret{
		Key:    "secret1",
		Secret: []byte("password"),
	}
	secret2 := &vault.Secret{
		Key:    "secret2",
		Secret: []byte("password2"),
	}
	secret3 := &vault.Secret{
		Key:    "secret3",
		Secret: []byte("password3"),
	}

	ctx := context.Background()
	var err error

	v1 := inmemory.New()
	_, err = v1.Set(ctx, secret1)
	require.NoError(t, err)

	v2 := inmemory.New()
	_, err = v2.Set(ctx, secret2)
	require.NoError(t, err)

	m := New(v1, v2)

	var sec *vault.Secret
	sec, err = m.Get(ctx, &vault.SecretID{
		Key: secret1.Key,
	})
	require.NoError(t, err)
	assert.Equal(t, secret1, sec)

	sec, err = m.Get(ctx, &vault.SecretID{
		Key: secret2.Key,
	})
	require.NoError(t, err)
	assert.Equal(t, secret2, sec)

	sec, err = m.Get(ctx, &vault.SecretID{
		Key: secret3.Key,
	})
	require.Error(t, err)
}
