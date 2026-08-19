// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	mondoovault "go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

func TestNormalizeEndpoint(t *testing.T) {
	cases := []struct {
		name       string
		in         string
		wantHost   string
		wantSecure bool
		wantErr    bool
	}{
		{"https url", "https://minio.example.com:9000", "minio.example.com:9000", true, false},
		{"http url", "http://minio.example.com:9000", "minio.example.com:9000", false, false},
		// A bare host defaults to HTTPS. Defaulting to plaintext would silently
		// send the access key in the clear on a deployment that serves TLS.
		{"bare host", "minio.example.com:9000", "minio.example.com:9000", true, false},
		{"bare host no port", "minio.example.com", "minio.example.com", true, false},
		{"trailing slash", "https://minio.example.com:9000/", "minio.example.com:9000", true, false},
		{"surrounding space", "  https://minio.example.com:9000  ", "minio.example.com:9000", true, false},
		{"ipv4", "http://127.0.0.1:9000", "127.0.0.1:9000", false, false},
		{"empty", "", "", false, true},
		{"blank", "   ", "", false, true},
		{"unsupported scheme", "ftp://minio.example.com", "", false, true},
		{"no host", "https://", "", false, true},
		// A path would be silently dropped by the S3 client and produce
		// requests against a different endpoint than the operator asked for.
		{"path", "https://minio.example.com/prefix", "", false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host, secure, err := NormalizeEndpoint(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantHost, host)
			assert.Equal(t, tc.wantSecure, secure)
		})
	}
}

func conf(options map[string]string) *inventory.Config {
	return &inventory.Config{Options: options}
}

func TestSkipVerifyFromConf(t *testing.T) {
	value, err := skipVerifyFromConf(nil)
	require.NoError(t, err)
	assert.False(t, value)

	value, err = skipVerifyFromConf(conf(map[string]string{}))
	require.NoError(t, err)
	assert.False(t, value)

	value, err = skipVerifyFromConf(conf(map[string]string{OptionTLSSkipVerify: "true"}))
	require.NoError(t, err)
	assert.True(t, value)

	value, err = skipVerifyFromConf(conf(map[string]string{OptionTLSSkipVerify: "false"}))
	require.NoError(t, err)
	assert.False(t, value)

	// A typo must be an error rather than a silent false, so an operator who
	// believes verification is off is not left with it quietly on, or vice
	// versa depending on which way they misread the result.
	_, err = skipVerifyFromConf(conf(map[string]string{OptionTLSSkipVerify: "yes please"}))
	require.Error(t, err)
}

func TestNewTransport(t *testing.T) {
	t.Run("plain endpoint never skips verification", func(t *testing.T) {
		transport, err := newTransport(conf(nil), false)
		require.NoError(t, err)
		httpTransport, ok := transport.(*http.Transport)
		require.True(t, ok)
		if httpTransport.TLSClientConfig != nil {
			assert.False(t, httpTransport.TLSClientConfig.InsecureSkipVerify)
			assert.Nil(t, httpTransport.TLSClientConfig.RootCAs)
		}
	})

	t.Run("TLS options on a plain endpoint are an error", func(t *testing.T) {
		// Accepting them silently would leave an operator believing they had
		// pinned an authority on a connection that carries no TLS at all.
		_, err := newTransport(conf(map[string]string{OptionTLSSkipVerify: "true"}), false)
		require.Error(t, err)

		_, err = newTransport(conf(map[string]string{OptionCACert: "/etc/ca.pem"}), false)
		require.Error(t, err)
	})

	t.Run("secure endpoint verifies by default", func(t *testing.T) {
		transport, err := newTransport(conf(nil), true)
		require.NoError(t, err)
		httpTransport := transport.(*http.Transport)
		require.NotNil(t, httpTransport.TLSClientConfig)
		assert.False(t, httpTransport.TLSClientConfig.InsecureSkipVerify)
		assert.Equal(t, uint16(tls.VersionTLS12), httpTransport.TLSClientConfig.MinVersion)
	})

	t.Run("skip verify is honored", func(t *testing.T) {
		transport, err := newTransport(conf(map[string]string{OptionTLSSkipVerify: "true"}), true)
		require.NoError(t, err)
		assert.True(t, transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("inline PEM is trusted", func(t *testing.T) {
		transport, err := newTransport(conf(map[string]string{OptionCACert: testCAPEM}), true)
		require.NoError(t, err)
		assert.NotNil(t, transport.(*http.Transport).TLSClientConfig.RootCAs)
	})

	t.Run("mangled PEM is an error, not a silent fallback", func(t *testing.T) {
		// Falling back to the system roots here would leave the connection
		// trusting a completely different set of authorities than the operator
		// asked for, and failing later for an unrelated-looking reason.
		_, err := newTransport(conf(map[string]string{
			OptionCACert: "-----BEGIN CERTIFICATE-----\nnot base64\n-----END CERTIFICATE-----",
		}), true)
		require.Error(t, err)
	})

	t.Run("missing CA file is an error", func(t *testing.T) {
		_, err := newTransport(conf(map[string]string{
			OptionCACert: "/nonexistent/path/to/ca.pem",
		}), true)
		require.Error(t, err)
	})

	t.Run("transports do not share configuration", func(t *testing.T) {
		a, err := newTransport(conf(map[string]string{OptionTLSSkipVerify: "true"}), true)
		require.NoError(t, err)
		b, err := newTransport(conf(nil), true)
		require.NoError(t, err)
		assert.True(t, a.(*http.Transport).TLSClientConfig.InsecureSkipVerify)
		assert.False(t, b.(*http.Transport).TLSClientConfig.InsecureSkipVerify,
			"one connection skipping verification must not affect another")
	})
}

func TestIsPEM(t *testing.T) {
	assert.True(t, isPEM("-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----"))
	assert.False(t, isPEM("/etc/ssl/certs/ca.pem"))
	assert.False(t, isPEM(""))
}

func TestCredentialsFromConf(t *testing.T) {
	accessKey, secretKey := credentialsFromConf(nil)
	assert.Empty(t, accessKey)
	assert.Empty(t, secretKey)

	accessKey, secretKey = credentialsFromConf(&inventory.Config{})
	assert.Empty(t, accessKey)
	assert.Empty(t, secretKey)

	accessKey, secretKey = credentialsFromConf(&inventory.Config{
		Credentials: []*mondoovault.Credential{
			nil,
			{Type: mondoovault.CredentialType_ssh_agent, User: "ignored", Secret: []byte("x")},
			{Type: mondoovault.CredentialType_password, User: "ACCESS", Secret: []byte("SECRET")},
		},
	})
	assert.Equal(t, "ACCESS", accessKey)
	assert.Equal(t, "SECRET", secretKey)

	// A credential with no secret is not a key pair.
	accessKey, secretKey = credentialsFromConf(&inventory.Config{
		Credentials: []*mondoovault.Credential{
			{Type: mondoovault.CredentialType_password, User: "ACCESS"},
		},
	})
	assert.Empty(t, accessKey)
	assert.Empty(t, secretKey)
}

func TestNewMinioConnectionRequiresCredentials(t *testing.T) {
	t.Setenv("MINIO_ENDPOINT", "")
	t.Setenv("MINIO_ROOT_USER", "")
	t.Setenv("MINIO_ROOT_PASSWORD", "")
	t.Setenv("MINIO_ACCESS_KEY", "")
	t.Setenv("MINIO_SECRET_KEY", "")

	_, err := NewMinioConnection(1, &inventory.Asset{}, conf(nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint")

	_, err = NewMinioConnection(1, &inventory.Asset{},
		conf(map[string]string{OptionEndpoint: "https://minio.example.com:9000"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access key")
}

func TestNewMinioConnectionReadsEnvironment(t *testing.T) {
	t.Setenv("MINIO_ENDPOINT", "http://minio.example.com:9000")
	t.Setenv("MINIO_ROOT_USER", "rootuser")
	t.Setenv("MINIO_ROOT_PASSWORD", "rootpass")

	conn, err := NewMinioConnection(1, &inventory.Asset{}, conf(nil))
	require.NoError(t, err)
	assert.Equal(t, "minio.example.com:9000", conn.Host())
	assert.False(t, conn.Secure())
	assert.NotNil(t, conn.Client())
	assert.NotNil(t, conn.Admin())
}

func TestNewMinioConnectionPrefersRootOverAccessKeyEnv(t *testing.T) {
	t.Setenv("MINIO_ENDPOINT", "https://minio.example.com:9000")
	t.Setenv("MINIO_ROOT_USER", "rootuser")
	t.Setenv("MINIO_ROOT_PASSWORD", "rootpass")
	t.Setenv("MINIO_ACCESS_KEY", "otheruser")
	t.Setenv("MINIO_SECRET_KEY", "otherpass")

	conn, err := NewMinioConnection(1, &inventory.Asset{}, conf(nil))
	require.NoError(t, err)
	assert.True(t, conn.Secure())
}

func TestNewMinioConnectionFallsBackToAccessKeyEnv(t *testing.T) {
	t.Setenv("MINIO_ENDPOINT", "https://minio.example.com:9000")
	t.Setenv("MINIO_ROOT_USER", "")
	t.Setenv("MINIO_ROOT_PASSWORD", "")
	t.Setenv("MINIO_ACCESS_KEY", "otheruser")
	t.Setenv("MINIO_SECRET_KEY", "otherpass")

	_, err := NewMinioConnection(1, &inventory.Asset{}, conf(nil))
	require.NoError(t, err)
}

func TestFirstNonEmpty(t *testing.T) {
	assert.Equal(t, "a", firstNonEmpty("a", "b"))
	assert.Equal(t, "b", firstNonEmpty("", "b"))
	assert.Equal(t, "", firstNonEmpty("", ""))
	assert.Equal(t, "", firstNonEmpty())
}

// testCAPEM is a self-signed certificate used only to prove the trust store is
// populated from operator-supplied material. It holds no private key.
const testCAPEM = `-----BEGIN CERTIFICATE-----
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
