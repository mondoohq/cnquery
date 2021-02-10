package keyring

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/vault"
	"gotest.tools/assert"
)

func TestKeyring(t *testing.T) {
	v := New("mondoo")
	ctx := context.Background()

	credSecret := map[string]string{
		"key":  "value",
		"key2": "value2",
	}
	credBytes, err := json.Marshal(credSecret)
	require.NoError(t, err)

	key := vault.Mrn2secretKey("//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/675173580680/regions/eu-west-1/instances/i-0e11b0762369fbefa")
	cred := &vault.Credential{
		Key:    key,
		Label:  "mondoo: " + key,
		Secret: string(credBytes),
	}

	id, err := v.Set(ctx, cred)
	require.NoError(t, err)

	newCred, err := v.Get(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, key, newCred.Key)
	assert.Equal(t, cred.Label, newCred.Label)
	assert.Equal(t, cred.Secret, newCred.Secret)
}
