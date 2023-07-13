package vault

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestVaultTypeParser(t *testing.T) {
	content := `
- none
- keyring
- linux-kernel-keyring
- encrypted-file
- hashicorp-vault
- gcp-secret-manager
- aws-secrets-manager
- aws-parameter-store
`

	v := []VaultType{}
	yaml.Unmarshal([]byte(content), &v)

	assert.Equal(t, 8, len(v))
	assert.Equal(t, VaultType_None, v[0])
	assert.Equal(t, VaultType_KeyRing, v[1])
	assert.Equal(t, VaultType_LinuxKernelKeyring, v[2])
	assert.Equal(t, VaultType_EncryptedFile, v[3])
	assert.Equal(t, VaultType_HashiCorp, v[4])
	assert.Equal(t, VaultType_GCPSecretsManager, v[5])
	assert.Equal(t, VaultType_AWSSecretsManager, v[6])
	assert.Equal(t, VaultType_AWSParameterStore, v[7])
}

func TestVaultTypeMarshal(t *testing.T) {
	data, err := json.Marshal(VaultType_LinuxKernelKeyring)
	require.NoError(t, err)
	assert.Equal(t, "\"linux-kernel-keyring\"", string(data))
}
