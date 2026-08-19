// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"net/http"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"

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
		{"https with port", "https://vault.example.com:8200", "https://vault.example.com:8200", "vault.example.com:8200", false},
		{"trailing slash trimmed", "https://vault.example.com:8200/", "https://vault.example.com:8200", "vault.example.com:8200", false},
		{"http preserved", "http://127.0.0.1:8200", "http://127.0.0.1:8200", "127.0.0.1:8200", false},
		// a bare host defaults to https rather than http
		{"bare host defaults to https", "vault.example.com:8200", "https://vault.example.com:8200", "vault.example.com:8200", false},
		{"whitespace trimmed", "  https://vault.example.com:8200  ", "https://vault.example.com:8200", "vault.example.com:8200", false},
		{"query and fragment dropped", "https://vault.example.com:8200/?a=b#c", "https://vault.example.com:8200", "vault.example.com:8200", false},
		{"empty", "", "", "", true},
		{"unsupported scheme", "ftp://vault.example.com", "", "", true},
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

func TestCredentialsFromConf(t *testing.T) {
	t.Run("bare password is the token", func(t *testing.T) {
		token, secretID := credentialsFromConf(&inventory.Config{
			Credentials: []*mondoovault.Credential{
				{Type: mondoovault.CredentialType_password, Secret: []byte("hvs.token")},
			},
		})
		assert.Equal(t, "hvs.token", token)
		assert.Empty(t, secretID)
	})

	t.Run("secret-id user tags the AppRole secret", func(t *testing.T) {
		token, secretID := credentialsFromConf(&inventory.Config{
			Credentials: []*mondoovault.Credential{
				{Type: mondoovault.CredentialType_password, User: "secret-id", Secret: []byte("s3cret")},
			},
		})
		assert.Empty(t, token)
		assert.Equal(t, "s3cret", secretID)
	})

	t.Run("both can be present at once", func(t *testing.T) {
		token, secretID := credentialsFromConf(&inventory.Config{
			Credentials: []*mondoovault.Credential{
				{Type: mondoovault.CredentialType_password, Secret: []byte("hvs.token")},
				{Type: mondoovault.CredentialType_password, User: "secret-id", Secret: []byte("s3cret")},
			},
		})
		assert.Equal(t, "hvs.token", token)
		assert.Equal(t, "s3cret", secretID)
	})

	t.Run("empty secrets and wrong types are ignored", func(t *testing.T) {
		token, secretID := credentialsFromConf(&inventory.Config{
			Credentials: []*mondoovault.Credential{
				nil,
				{Type: mondoovault.CredentialType_password, Secret: []byte{}},
				{Type: mondoovault.CredentialType_private_key, Secret: []byte("key")},
			},
		})
		assert.Empty(t, token)
		assert.Empty(t, secretID)
	})

	t.Run("nil config", func(t *testing.T) {
		token, secretID := credentialsFromConf(nil)
		assert.Empty(t, token)
		assert.Empty(t, secretID)
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
	assert.False(t, isPEM("/etc/ssl/certs/ca.pem"))
	assert.False(t, isPEM(""))
}

func TestNewVaultServerIdentifier(t *testing.T) {
	// a colon is legal inside a path segment, so the host stays readable
	assert.Equal(t,
		PlatformIdVaultServer+"vault.example.com:8200",
		NewVaultServerIdentifier("vault.example.com:8200", ""))

	// a namespace carries slashes, which must not read as a deeper path
	assert.Equal(t,
		PlatformIdVaultServer+"vault.example.com:8200/namespace/team-a%2Fsub",
		NewVaultServerIdentifier("vault.example.com:8200", "team-a/sub"))

	// two namespaces on one host must not collide
	assert.NotEqual(t,
		NewVaultServerIdentifier("vault.example.com:8200", "team-a"),
		NewVaultServerIdentifier("vault.example.com:8200", "team-b"))
}

func TestNewVaultServerPlatformAlwaysCarriesNamespace(t *testing.T) {
	root := NewVaultServerPlatform("vault.example.com:8200", "")
	assert.Equal(t,
		[]string{"saas", "vault", "host", "vault.example.com:8200", "namespace", RootNamespaceLabel},
		root.TechnologyUrlSegments)

	scoped := NewVaultServerPlatform("vault.example.com:8200", "team-a/")
	assert.Equal(t,
		[]string{"saas", "vault", "host", "vault.example.com:8200", "namespace", "team-a/"},
		scoped.TechnologyUrlSegments)
}

func TestApplyTLSConfig(t *testing.T) {
	// a self-signed authority, used only to prove valid PEM is accepted
	const caPEM = `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`

	t.Run("inline PEM is trusted without leaving a temp file", func(t *testing.T) {
		cfg := vaultapi.DefaultConfig()
		require.NoError(t, cfg.Error)
		err := applyTLSConfig(cfg, &inventory.Config{
			Options: map[string]string{OptionCACert: caPEM},
		})
		require.NoError(t, err)

		transport, ok := cfg.HttpClient.Transport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, transport.TLSClientConfig)
		assert.NotNil(t, transport.TLSClientConfig.RootCAs)
	})

	// a mangled certificate must not leave the connection quietly trusting
	// only the system roots and failing later for an unrelated-looking reason
	t.Run("invalid inline PEM is an error, not a no-op", func(t *testing.T) {
		cfg := vaultapi.DefaultConfig()
		require.NoError(t, cfg.Error)
		err := applyTLSConfig(cfg, &inventory.Config{
			Options: map[string]string{OptionCACert: "-----BEGIN CERTIFICATE-----\nnot base64\n-----END CERTIFICATE-----"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), OptionCACert)
	})

	t.Run("skip verify is applied", func(t *testing.T) {
		cfg := vaultapi.DefaultConfig()
		require.NoError(t, cfg.Error)
		require.NoError(t, applyTLSConfig(cfg, &inventory.Config{
			Options: map[string]string{OptionTLSSkipVerify: "true"},
		}))

		transport, ok := cfg.HttpClient.Transport.(*http.Transport)
		require.True(t, ok)
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("verification stays on by default", func(t *testing.T) {
		cfg := vaultapi.DefaultConfig()
		require.NoError(t, cfg.Error)
		require.NoError(t, applyTLSConfig(cfg, &inventory.Config{}))

		transport, ok := cfg.HttpClient.Transport.(*http.Transport)
		require.True(t, ok)
		assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)
	})
}
