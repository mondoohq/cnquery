package keyring

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/vault"
)

func TestKeyring(t *testing.T) {
	v := New("mondoo")
	ctx := context.Background()

	key := vault.Mrn2secretKey("//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/675173580680/regions/eu-west-1/instances/i-0e11b0762369fbefa")
	cred := &vault.Credential{
		Key:   key,
		Label: "mondoo: " + key,
		Fields: map[string]string{
			"key":  "value",
			"key2": "value2",
		},
	}

	id, err := v.Set(ctx, cred)
	require.NoError(t, err)

	newCred, err := v.Get(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, cred, newCred)
}
