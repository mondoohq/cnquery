// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	mondoovault "go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantAddr string
		wantHost string
		wantErr  bool
	}{
		{"plain https", "https://rancher.example.com", "https://rancher.example.com", "rancher.example.com", false},
		{"trailing slash", "https://rancher.example.com/", "https://rancher.example.com", "rancher.example.com", false},
		{"with port", "https://rancher.example.com:8443", "https://rancher.example.com:8443", "rancher.example.com:8443", false},
		{"bare host defaults to https", "rancher.example.com", "https://rancher.example.com", "rancher.example.com", false},
		{"api suffix trimmed", "https://rancher.example.com/v3", "https://rancher.example.com", "rancher.example.com", false},
		{"steve suffix trimmed", "https://rancher.example.com/v1/", "https://rancher.example.com", "rancher.example.com", false},
		{"query dropped", "https://rancher.example.com/?x=1", "https://rancher.example.com", "rancher.example.com", false},
		{"sub path kept", "https://proxy.example.com/rancher", "https://proxy.example.com/rancher", "proxy.example.com", false},
		{"plaintext allowed for a lab server", "http://localhost:8080", "http://localhost:8080", "localhost:8080", false},
		{"empty", "", "", "", true},
		{"blank", "   ", "", "", true},
		{"unsupported scheme", "ftp://rancher.example.com", "", "", true},
		{"scheme with no host", "https://", "", "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			address, host, err := NormalizeURL(test.input)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantAddr, address)
			assert.Equal(t, test.wantHost, host)
		})
	}
}

func passwordCredential(user, secret string) *mondoovault.Credential {
	return &mondoovault.Credential{
		Type:   mondoovault.CredentialType_password,
		User:   user,
		Secret: []byte(secret),
	}
}

func TestResolveToken(t *testing.T) {
	t.Setenv("RANCHER_TOKEN", "")
	t.Setenv("RANCHER_ACCESS_KEY", "")
	t.Setenv("RANCHER_SECRET_KEY", "")

	t.Run("whole token passes through", func(t *testing.T) {
		conf := &inventory.Config{
			Credentials: []*mondoovault.Credential{passwordCredential("", "token-abcde:s3cr3t")},
		}
		token, err := resolveToken(conf)
		require.NoError(t, err)
		assert.Equal(t, "token-abcde:s3cr3t", token)
	})

	t.Run("access key and secret key are joined", func(t *testing.T) {
		conf := &inventory.Config{
			Options:     map[string]string{OptionAccessKey: "token-abcde"},
			Credentials: []*mondoovault.Credential{passwordCredential("secret-key", "s3cr3t")},
		}
		token, err := resolveToken(conf)
		require.NoError(t, err)
		assert.Equal(t, "token-abcde:s3cr3t", token)
	})

	t.Run("access key without a secret key is refused", func(t *testing.T) {
		conf := &inventory.Config{Options: map[string]string{OptionAccessKey: "token-abcde"}}
		_, err := resolveToken(conf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret key")
	})

	t.Run("no credential at all is refused", func(t *testing.T) {
		_, err := resolveToken(&inventory.Config{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API token")
	})

	t.Run("environment fallback", func(t *testing.T) {
		t.Setenv("RANCHER_TOKEN", "token-env:secret")
		token, err := resolveToken(&inventory.Config{})
		require.NoError(t, err)
		assert.Equal(t, "token-env:secret", token)
	})
}

func TestSkipVerifyFromConf(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    bool
		wantErr bool
	}{
		{"unset", "", false, false},
		{"true", "true", true, false},
		{"false", "false", false, false},
		{"one", "1", true, false},
		// A typo must not read as "verification stays on" silently, because the
		// operator believes they turned it off and would not look again.
		{"typo", "ture", false, true},
		{"yes is not a bool here", "yes", false, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conf := &inventory.Config{Options: map[string]string{}}
			if test.value != "" {
				conf.Options[OptionTLSSkipVerify] = test.value
			}
			got, err := skipVerifyFromConf(conf)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestCertPoolRejectsBadMaterial(t *testing.T) {
	// A mangled certificate must fail rather than silently leaving the
	// connection trusting only the system roots.
	_, err := certPool("-----BEGIN CERTIFICATE-----\nnot base64\n-----END CERTIFICATE-----")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid PEM")

	_, err = certPool("/nonexistent/path/to/ca.pem")
	require.Error(t, err)
}

func TestIsPEM(t *testing.T) {
	assert.True(t, isPEM("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"))
	assert.False(t, isPEM("/etc/ssl/certs/rancher-ca.pem"))
}

func TestNewRancherServerIdentifier(t *testing.T) {
	assert.Equal(t,
		PlatformIdRancherServer+"rancher.example.com",
		NewRancherServerIdentifier("rancher.example.com"))
	// A port makes a different install on the same host, and the identifier
	// must keep them apart.
	assert.NotEqual(t,
		NewRancherServerIdentifier("rancher.example.com"),
		NewRancherServerIdentifier("rancher.example.com:8443"))
}

func TestNewRancherServerPlatform(t *testing.T) {
	platform := NewRancherServerPlatform("rancher.example.com")
	require.NotNil(t, platform)
	assert.Equal(t, "rancher", platform.Name)
	assert.Contains(t, platform.TechnologyUrlSegments, "rancher")
	assert.Contains(t, platform.TechnologyUrlSegments, "rancher.example.com")
}
