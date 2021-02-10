package discovery

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	mockvault "go.mondoo.io/mondoo/motor/vault/mock"

	"go.mondoo.io/mondoo/motor/transports"
	"gotest.tools/assert"
)

func TestParseJsonByFields(t *testing.T) {
	connection := transports.TransportConfig{
		Backend:  transports.TransportBackend_CONNECTION_SSH,
		Insecure: true,
	}
	secretInfo := secretInfo{
		secretFormat: "json",
		jsonFields:   []string{"user", "password"},
	}
	test := `{"password":"pass","user":"ec2"}`

	err := parseJsonByFields([]byte(test), &secretInfo, &connection)
	require.NoError(t, err)
	assert.Equal(t, "ec2", connection.User)
	assert.Equal(t, "pass", connection.Password)

	test = `{"private_key":"blabla","user":"some-user"}`
	secretInfo.jsonFields = []string{"user", "private_key"}
	err = parseJsonByFields([]byte(test), &secretInfo, &connection)
	require.NoError(t, err)
	assert.Equal(t, "some-user", connection.User)
	assert.Equal(t, "blabla", string(connection.PrivateKeyBytes))
}

func TestEnrichAssetWithVaultData(t *testing.T) {
	v := mockvault.New()
	connections := []*transports.TransportConfig{
		{Backend: transports.TransportBackend_CONNECTION_SSH, Insecure: true},
	}
	a := asset.Asset{
		Name:        "asset-name",
		Platform:    &platform.Platform{Name: "ubuntu"},
		Connections: connections,
	}
	secretInfo := secretInfo{
		user:         "test-user",
		secretID:     "mockPassword",
		secretFormat: "password",
	}

	enrichAssetWithVaultData(v, &a, &secretInfo)
	assert.Equal(t, "asset-name", a.Name)
	assert.Equal(t, "test-user", a.Connections[0].User)
	assert.Equal(t, mockvault.MockPassword, a.Connections[0].Password)

	secretInfo.user = "some-user"
	secretInfo.secretID = "mockPKey"
	secretInfo.secretFormat = "private_key"

	enrichAssetWithVaultData(v, &a, &secretInfo)
	assert.Equal(t, "asset-name", a.Name)
	assert.Equal(t, "some-user", a.Connections[0].User)
	assert.Equal(t, mockvault.MockPKey, string(a.Connections[0].PrivateKeyBytes))

	secretInfo.secretID = "mockJson"
	secretInfo.secretFormat = "json"
	secretInfo.jsonFields = []string{"user", "private_key"}

	enrichAssetWithVaultData(v, &a, &secretInfo)
	assert.Equal(t, "asset-name", a.Name)
	assert.Equal(t, "that-user", a.Connections[0].User)
	assert.Equal(t, "blabla", string(a.Connections[0].PrivateKeyBytes))

	secretInfo.secretID = "bad-id"
	secretInfo.secretFormat = "json"
	secretInfo.jsonFields = []string{"user", "private_key"}

	enrichAssetWithVaultData(v, &a, &secretInfo)
	assert.Equal(t, "asset-name", a.Name)
	assert.Equal(t, "that-user", a.Connections[0].User)
	assert.Equal(t, "blabla", string(a.Connections[0].PrivateKeyBytes))

}
