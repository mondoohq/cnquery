// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/hcp/connection"
)

func parseCLI(t *testing.T, flags map[string]string) *inventory.Config {
	t.Helper()
	primitives := map[string]*llx.Primitive{}
	for k, v := range flags {
		primitives[k] = llx.StringPrimitive(v)
	}
	res, err := Init().ParseCLI(&plugin.ParseCLIReq{Connector: "hcp", Flags: primitives})
	require.NoError(t, err)
	require.NotNil(t, res.Asset)
	require.Len(t, res.Asset.Connections, 1)
	return res.Asset.Connections[0]
}

func credential(conf *inventory.Config, user string) string {
	for _, cred := range conf.Credentials {
		if cred.Type == vault.CredentialType_password && cred.User == user {
			return string(cred.Secret)
		}
	}
	return ""
}

func TestParseCLICarriesTerraformSettings(t *testing.T) {
	// The token must land under its own credential tag, not the service
	// principal's: a crossed tag would authenticate the wrong control plane.
	conf := parseCLI(t, map[string]string{
		"client-id":        "cid",
		"client-secret":    "csecret",
		"tfe-token":        "tfe-token-value",
		"tfe-address":      "https://tfe.example.com",
		"tfe-organization": "acme",
	})

	assert.Equal(t, "cid", conf.Options[connection.OptionClientID])
	assert.Equal(t, "csecret", credential(conf, connection.CredentialClientSecret))
	assert.Equal(t, "tfe-token-value", credential(conf, connection.CredentialTfeToken))
	assert.Equal(t, "https://tfe.example.com", conf.Options[connection.OptionTfeAddress])
	assert.Equal(t, "acme", conf.Options[connection.OptionTfeOrganization])
}

func TestParseCLIOmitsUnsetTerraformSettings(t *testing.T) {
	conf := parseCLI(t, map[string]string{"client-id": "cid", "client-secret": "csecret"})

	assert.Equal(t, "", credential(conf, connection.CredentialTfeToken))
	_, hasAddress := conf.Options[connection.OptionTfeAddress]
	assert.False(t, hasAddress)
	_, hasOrg := conf.Options[connection.OptionTfeOrganization]
	assert.False(t, hasOrg)
}

func TestParseCLIReadsTerraformEnvironment(t *testing.T) {
	t.Setenv("TFE_TOKEN", "env-token")
	t.Setenv("TFE_ADDRESS", "https://tfe.env.example.com")
	t.Setenv("TFE_ORGANIZATION", "env-org")

	conf := parseCLI(t, map[string]string{})
	assert.Equal(t, "env-token", credential(conf, connection.CredentialTfeToken))
	assert.Equal(t, "https://tfe.env.example.com", conf.Options[connection.OptionTfeAddress])
	assert.Equal(t, "env-org", conf.Options[connection.OptionTfeOrganization])

	// An explicit flag wins over the environment.
	conf = parseCLI(t, map[string]string{"tfe-token": "flag-token"})
	assert.Equal(t, "flag-token", credential(conf, connection.CredentialTfeToken))
}
