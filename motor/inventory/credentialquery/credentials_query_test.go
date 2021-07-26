package credentialquery

import (
	"testing"

	"go.mondoo.io/mondoo/types"

	"github.com/stretchr/testify/assert"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/policy/executor"
)

func TestSecretKeySimple(t *testing.T) {
	query := `{type: 'ssh_agent'}`

	e, err := executor.NewEmbeddedExecutor()
	require.NoError(t, err)

	value, err := e.Run(query, map[string]*llx.Primitive{})
	require.NoError(t, err)

	sMeta := &CredentialQueryResponse{}
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   sMeta,
		TagName:  "json",
	})
	err = decoder.Decode(value)
	require.NoError(t, err)
	assert.Equal(t, "ssh_agent", sMeta.Type)
}

func TestSecretKeyIfReturn(t *testing.T) {
	e, err := executor.NewEmbeddedExecutor()
	require.NoError(t, err)

	query := `
		if (props.a == 'windows' && props.labels['key'] == 'value') {
			return {type: 'password', secret_id: 'theonekey'}
		}
		return {type: 'password', secret_id: 'otherkey'}
	`

	props := map[string]*llx.Primitive{
		"a": llx.StringPrimitive("windows"),
		"labels": llx.MapData(map[string]interface{}{
			"key": "value",
		}, types.String).Result().Data,
	}

	value, err := e.Run(query, props)
	require.NoError(t, err)

	sMeta := &CredentialQueryResponse{}
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   sMeta,
		TagName:  "json",
	})
	err = decoder.Decode(value)
	require.NoError(t, err)

	// NOTE: this is not working yet
	assert.Equal(t, "password", sMeta.Type)
	assert.Equal(t, "theonekey", sMeta.SecretId)
}
