package credentialquery

import (
	"testing"

	"go.mondoo.io/mondoo/motor/vault"

	"go.mondoo.io/mondoo/motor/asset"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestSecretKeySimple(t *testing.T) {
	query := `{ type: 'ssh_agent' }`

	runner, err := NewCredentialQueryRunner(query)
	require.NoError(t, err)
	cred, err := runner.Run(&asset.Asset{})
	require.NoError(t, err)
	assert.Equal(t, vault.CredentialType_ssh_agent, cred.Type)
}

func TestSecretKeyIfReturn(t *testing.T) {
	query := `
		if (props.labels['key'] == 'value') {
			return {type: 'password', secret_id: 'theonekey'}
		}
		return {type: 'private_key', secret_id: 'otherkey'}
	`

	runner, err := NewCredentialQueryRunner(query)
	require.NoError(t, err)

	cred, err := runner.Run(&asset.Asset{
		Labels: map[string]string{
			"key": "value",
		},
	})
	require.NoError(t, err)

	assert.Equal(t, vault.CredentialType_password, cred.Type)
	assert.Equal(t, "theonekey", cred.SecretId)
}
