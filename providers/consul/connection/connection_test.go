// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"crypto/tls"
	"net/http"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	mondoovault "go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

func TestNormalizeAddress(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		wantAddress string
		wantHost    string
		wantErr     bool
	}{
		{"http with port", "http://127.0.0.1:8500", "http://127.0.0.1:8500", "127.0.0.1:8500", false},
		{"https with port", "https://consul.example.com:8501", "https://consul.example.com:8501", "consul.example.com:8501", false},
		{"trailing slash trimmed", "http://127.0.0.1:8500/", "http://127.0.0.1:8500", "127.0.0.1:8500", false},
		// the Consul client and CLI both assume plaintext for a bare address,
		// and that is what an unconfigured agent serves
		{"bare host defaults to http", "consul.example.com:8500", "http://consul.example.com:8500", "consul.example.com:8500", false},
		// two spellings of one agent must not become two assets
		{"http port filled in", "http://consul.example.com", "http://consul.example.com:8500", "consul.example.com:8500", false},
		{"https port filled in", "https://consul.example.com", "https://consul.example.com:8501", "consul.example.com:8501", false},
		{"whitespace trimmed", "  http://127.0.0.1:8500  ", "http://127.0.0.1:8500", "127.0.0.1:8500", false},
		{"query and fragment dropped", "http://127.0.0.1:8500/?a=b#c", "http://127.0.0.1:8500", "127.0.0.1:8500", false},
		{"empty", "", "", "", true},
		// a unix socket carries no host to build an asset identity from
		{"unix socket refused", "unix:///var/run/consul.sock", "", "", true},
		{"unsupported scheme", "ftp://consul.example.com", "", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			address, host, err := NormalizeAddress(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantAddress, address)
			assert.Equal(t, tc.wantHost, host)
		})
	}
}

func TestNormalizeAddressAgreesOnOneAgent(t *testing.T) {
	// the default address and its explicit spelling must produce one identity
	withPort, hostWithPort, err := NormalizeAddress(DefaultAddress)
	require.NoError(t, err)
	withoutPort, hostWithoutPort, err := NormalizeAddress("127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, withPort, withoutPort)
	assert.Equal(t, hostWithPort, hostWithoutPort)
}

func TestTokenFromConf(t *testing.T) {
	t.Run("password credential is the token", func(t *testing.T) {
		token := tokenFromConf(&inventory.Config{
			Credentials: []*mondoovault.Credential{
				{Type: mondoovault.CredentialType_password, Secret: []byte("acl-token")},
			},
		})
		assert.Equal(t, "acl-token", token)
	})

	t.Run("empty secrets and wrong types are ignored", func(t *testing.T) {
		token := tokenFromConf(&inventory.Config{
			Credentials: []*mondoovault.Credential{
				nil,
				{Type: mondoovault.CredentialType_password, Secret: []byte{}},
				{Type: mondoovault.CredentialType_private_key, Secret: []byte("key")},
			},
		})
		assert.Empty(t, token)
	})

	t.Run("nil config", func(t *testing.T) {
		assert.Empty(t, tokenFromConf(nil))
	})
}

func TestSkipVerifyFromConf(t *testing.T) {
	t.Run("absent means verify", func(t *testing.T) {
		value, err := skipVerifyFromConf(&inventory.Config{})
		require.NoError(t, err)
		assert.False(t, value)
	})

	t.Run("true", func(t *testing.T) {
		value, err := skipVerifyFromConf(&inventory.Config{
			Options: map[string]string{OptionTLSSkipVerify: "true"},
		})
		require.NoError(t, err)
		assert.True(t, value)
	})

	// a typo must fail loudly rather than silently leaving verification on,
	// because the operator would believe they had disabled it
	t.Run("unparseable value is an error", func(t *testing.T) {
		_, err := skipVerifyFromConf(&inventory.Config{
			Options: map[string]string{OptionTLSSkipVerify: "yes-please"},
		})
		require.Error(t, err)
	})
}

func TestIsPEM(t *testing.T) {
	assert.True(t, isPEM("-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----"))
	assert.False(t, isPEM("/etc/consul.d/tls/ca.pem"))
	assert.False(t, isPEM(""))
}

func TestApplyTLSConfig(t *testing.T) {
	const caPEM = "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"

	t.Run("a path is passed through as a path", func(t *testing.T) {
		cfg := consulapi.DefaultConfig()
		require.NoError(t, applyTLSConfig(cfg, &inventory.Config{
			Options: map[string]string{OptionCACert: "/etc/consul.d/tls/ca.pem"},
		}))
		assert.Equal(t, "/etc/consul.d/tls/ca.pem", cfg.TLSConfig.CAFile)
		assert.Empty(t, cfg.TLSConfig.CAPem)
	})

	// inline material must not be written to a temporary file that would then
	// be left behind for every connection
	t.Run("inline material is passed through as material", func(t *testing.T) {
		cfg := consulapi.DefaultConfig()
		require.NoError(t, applyTLSConfig(cfg, &inventory.Config{
			Options: map[string]string{OptionCACert: caPEM},
		}))
		assert.Equal(t, []byte(caPEM), cfg.TLSConfig.CAPem)
		assert.Empty(t, cfg.TLSConfig.CAFile, "material must not be mistaken for a path")
	})

	t.Run("skip verify is applied", func(t *testing.T) {
		cfg := consulapi.DefaultConfig()
		require.NoError(t, applyTLSConfig(cfg, &inventory.Config{
			Options: map[string]string{OptionTLSSkipVerify: "true"},
		}))
		assert.True(t, cfg.TLSConfig.InsecureSkipVerify)
	})

	t.Run("verification stays on by default", func(t *testing.T) {
		cfg := consulapi.DefaultConfig()
		require.NoError(t, applyTLSConfig(cfg, &inventory.Config{}))
		assert.False(t, cfg.TLSConfig.InsecureSkipVerify)
	})

	t.Run("a bad skip-verify value stops the connection", func(t *testing.T) {
		cfg := consulapi.DefaultConfig()
		require.Error(t, applyTLSConfig(cfg, &inventory.Config{
			Options: map[string]string{OptionTLSSkipVerify: "maybe"},
		}))
	})

	// the settings must actually reach the transport the client dials with,
	// not merely sit in the configuration struct
	t.Run("the transport honors skip verify", func(t *testing.T) {
		cfg := consulapi.DefaultConfig()
		require.NoError(t, applyTLSConfig(cfg, &inventory.Config{
			Options: map[string]string{OptionTLSSkipVerify: "true"},
		}))
		httpClient, err := consulapi.NewHttpClient(cfg.Transport, cfg.TLSConfig)
		require.NoError(t, err)

		transport, ok := httpClient.Transport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, transport.TLSClientConfig)
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("the transport verifies by default", func(t *testing.T) {
		cfg := consulapi.DefaultConfig()
		require.NoError(t, applyTLSConfig(cfg, &inventory.Config{}))
		httpClient, err := consulapi.NewHttpClient(cfg.Transport, cfg.TLSConfig)
		require.NoError(t, err)

		transport, ok := httpClient.Transport.(*http.Transport)
		require.True(t, ok)
		var empty tls.Config
		if transport.TLSClientConfig != nil {
			assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)
		} else {
			assert.False(t, empty.InsecureSkipVerify)
		}
	})
}

func TestNewConsulAgentIdentifier(t *testing.T) {
	// a colon is legal inside a path segment, so the host stays readable
	assert.Equal(t,
		PlatformIdConsulAgent+"consul.example.com:8500",
		NewConsulAgentIdentifier("consul.example.com:8500"))

	// two agents in one datacenter carry their own TLS and gossip settings, so
	// they must not collapse into one asset
	assert.NotEqual(t,
		NewConsulAgentIdentifier("a.example.com:8500"),
		NewConsulAgentIdentifier("b.example.com:8500"))
}

func TestNewConsulAgentPlatformAlwaysCarriesDatacenter(t *testing.T) {
	known := NewConsulAgentPlatform("consul.example.com:8500", "dc1")
	assert.Equal(t,
		[]string{"saas", "consul", "host", "consul.example.com:8500", "datacenter", "dc1"},
		known.TechnologyUrlSegments)
	assert.Equal(t, "consul", known.Name)

	// the asset URL tree has a fixed depth, so a datacenter that could not be
	// read needs a label rather than an empty segment
	unknown := NewConsulAgentPlatform("consul.example.com:8500", "")
	assert.Equal(t,
		[]string{"saas", "consul", "host", "consul.example.com:8500", "datacenter", UnknownDatacenterLabel},
		unknown.TechnologyUrlSegments)
}
