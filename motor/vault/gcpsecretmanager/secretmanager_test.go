// +build debugtest

package gcpsecretmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/vault"
)

func TestGcpSecretmanager(t *testing.T) {
	projectID := "mondoo-dev-262313"
	v := New(projectID)
	ctx := context.Background()

	key := "mondoo-test-secret-key"
	cred := &vault.Secret{
		Key:    key,
		Secret: []byte("super-secret"),
	}

	id, err := v.Set(ctx, cred)
	require.NoError(t, err)

	newCred, err := v.Get(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, cred, newCred)
}
