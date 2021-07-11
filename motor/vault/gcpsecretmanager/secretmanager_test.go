// +build debugtest

package gcpsecretmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/vault"
)

func TestGcpSecretmanager(t *testing.T) {
	projectID := "mondoo-dev-262313"
	v := New(projectID)
	ctx := context.Background()

	cred := &vault.Secret{
		Key:    vault.Mrn2secretKey("//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/675173580680/regions/eu-west-1/instances/i-0e11b0762369fbefa"),
		Secret: []byte("super-secret"),
	}

	id, err := v.Set(ctx, cred)
	require.NoError(t, err)

	newCred, err := v.Get(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, cred, newCred)
}
