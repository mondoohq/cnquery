package inventory_test

import (
	"testing"

	"go.mondoo.io/mondoo/motor/inventory"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	mockvault "go.mondoo.io/mondoo/motor/vault/mock"
	"go.mondoo.io/mondoo/policy/executor"
	"go.mondoo.io/mondoo/types"
)

func TestSecretKeySimple(t *testing.T) {
	query := `{backend: 'ssh'}`

	e, err := executor.NewEmbeddedExecutor()
	require.NoError(t, err)

	value, err := e.Run(query, map[string]*llx.Primitive{})
	require.NoError(t, err)

	sMeta := &inventory.SecretMetadata{}
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   sMeta,
		TagName:  "json",
	})
	err = decoder.Decode(value)
	require.NoError(t, err)
	assert.Equal(t, "ssh", sMeta.Backend)
}

func TestSecretKeyIfReturn(t *testing.T) {
	e, err := executor.NewEmbeddedExecutor()
	require.NoError(t, err)

	query := `
		if (props.a == 'windows' && props.labels['key'] == 'value') {
			return {backend: 'ssh', secretID: 'theonekey'}
		}
		return {backend: 'ssh', secretID: 'otherkey'}
	`

	props := map[string]*llx.Primitive{
		"a": llx.StringPrimitive("windows"),
		"labels": llx.MapData(map[string]interface{}{
			"key": "value",
		}, types.String).Result().Data,
	}

	value, err := e.Run(query, props)
	require.NoError(t, err)

	sMeta := &inventory.SecretMetadata{}
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   sMeta,
		TagName:  "json",
	})
	err = decoder.Decode(value)
	require.NoError(t, err)

	// NOTE: this is not working yet
	assert.Equal(t, "ssh", sMeta.Backend)
	assert.Equal(t, "theonekey", sMeta.SecretID)
}

func TestSecretManagerPassword(t *testing.T) {
	v := mockvault.New()
	secretMetdataQuery := "{backend: 'ssh', secretFormat: 'password', secretID: 'mockPassword', user: 'test-user'}"
	vsm, err := inventory.NewVaultSecretManager(v, secretMetdataQuery)
	require.NoError(t, err)

	assetObj := &asset.Asset{
		Name:     "asset-name",
		Platform: &platform.Platform{Name: "ubuntu"},
		Connections: []*transports.TransportConfig{
			{Backend: transports.TransportBackend_CONNECTION_SSH, Insecure: true},
		},
	}
	secMeta, err := vsm.GetSecretMetadata(assetObj)
	require.NoError(t, err)

	// enrich connection with secret information
	err = vsm.EnrichConnection(assetObj, secMeta)
	require.NoError(t, err)

	assert.Equal(t, transports.TransportBackend_CONNECTION_SSH, assetObj.Connections[0].Backend)
	assert.Equal(t, "test-user", assetObj.Connections[0].User)
	assert.Equal(t, "password", assetObj.Connections[0].Password)
}

func TestSecretManagerPrivateKey(t *testing.T) {
	v := mockvault.New()
	secretMetdataQuery := "{backend: 'ssh', secretFormat: 'private_key',  secretID: 'mockPKey', user: 'some-user'}"
	vsm, err := inventory.NewVaultSecretManager(v, secretMetdataQuery)
	require.NoError(t, err)

	assetObj := &asset.Asset{
		Name:     "asset-name",
		Platform: &platform.Platform{Name: "ubuntu"},
		Connections: []*transports.TransportConfig{
			{Backend: transports.TransportBackend_CONNECTION_SSH, Insecure: true},
		},
	}
	secMeta, err := vsm.GetSecretMetadata(assetObj)
	require.NoError(t, err)

	// enrich connection with secret information
	err = vsm.EnrichConnection(assetObj, secMeta)
	require.NoError(t, err)

	assert.Equal(t, transports.TransportBackend_CONNECTION_SSH, assetObj.Connections[0].Backend)
	assert.Equal(t, "some-user", assetObj.Connections[0].User)
	assert.Equal(t, "", assetObj.Connections[0].Password)
	assert.Equal(t, []byte(mockvault.MockPKey), assetObj.Connections[0].PrivateKeyBytes)
}

func TestSecretManagerJSON(t *testing.T) {
	v := mockvault.New()
	secretMetdataQuery := "{secretFormat: 'json', secretID: 'mockJson'}"
	vsm, err := inventory.NewVaultSecretManager(v, secretMetdataQuery)
	require.NoError(t, err)

	assetObj := &asset.Asset{}
	secMeta, err := vsm.GetSecretMetadata(assetObj)
	require.NoError(t, err)

	// enrich connection with secret information
	err = vsm.EnrichConnection(assetObj, secMeta)
	require.NoError(t, err)

	assert.Equal(t, transports.TransportBackend_CONNECTION_SSH, assetObj.Connections[0].Backend)
	assert.Equal(t, "that-user", assetObj.Connections[0].User)
	assert.Equal(t, []byte("blabla"), assetObj.Connections[0].PrivateKeyBytes)
	assert.Equal(t, "supersecure", assetObj.Connections[0].Password)
}

func TestSecretManagerJSONBackendOverride(t *testing.T) {
	v := mockvault.New()
	secretMetdataQuery := "{backend: 'winrm', secretFormat: 'json', secretID: 'mockJson'}"
	vsm, err := inventory.NewVaultSecretManager(v, secretMetdataQuery)
	require.NoError(t, err)

	assetObj := &asset.Asset{}
	secMeta, err := vsm.GetSecretMetadata(assetObj)
	require.NoError(t, err)

	// enrich connection with secret information
	err = vsm.EnrichConnection(assetObj, secMeta)
	require.NoError(t, err)

	assert.Equal(t, transports.TransportBackend_CONNECTION_WINRM, assetObj.Connections[0].Backend)
	assert.Equal(t, "that-user", assetObj.Connections[0].User)
	assert.Equal(t, []byte("blabla"), assetObj.Connections[0].PrivateKeyBytes)
	assert.Equal(t, "supersecure", assetObj.Connections[0].Password)
}

func TestSecretManagerBadKey(t *testing.T) {
	v := mockvault.New()
	secretMetdataQuery := "{backend: 'ssh', secretFormat: 'json', secretID: 'bad-id'}"
	vsm, err := inventory.NewVaultSecretManager(v, secretMetdataQuery)
	require.NoError(t, err)

	assetObj := &asset.Asset{}
	secMeta, err := vsm.GetSecretMetadata(assetObj)
	require.NoError(t, err)

	// enrich connection with secret information
	err = vsm.EnrichConnection(assetObj, secMeta)
	require.Error(t, err)
}
