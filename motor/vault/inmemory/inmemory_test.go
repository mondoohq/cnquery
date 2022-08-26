package inmemory

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/vault"
	"gotest.tools/assert"
)

func TestVault(t *testing.T) {
	v := New()
	ctx := context.Background()

	credSecret := map[string]string{
		"key":  "value",
		"key2": "value2",
	}
	credBytes, err := json.Marshal(credSecret)
	require.NoError(t, err)

	key := "mondoo-test-secret-key"
	cred := &vault.Secret{
		Key:      key,
		Label:    "mondoo: " + key,
		Data:     credBytes,
		Encoding: vault.SecretEncoding_encoding_proto,
	}

	id, err := v.Set(ctx, cred)
	require.NoError(t, err)

	newCred, err := v.Get(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, key, newCred.Key)
	assert.Equal(t, cred.Label, newCred.Label)
	assert.DeepEqual(t, cred.Data, newCred.Data)
}
