// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"path/filepath"
	"strings"
	"testing"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	home          = getHomeDir()
	homeConfigDir = filepath.Join(home, ".config", "mondoo")
	homeConfig    = filepath.Join(homeConfigDir, DefaultConfigFile)

	systemConfigDir = filepath.Join("/etc", "opt", "mondoo")
	systemConfig    = filepath.Join(systemConfigDir, DefaultConfigFile)
	systemInventory = filepath.Join(systemConfigDir, "inventory.yml")

	oldConfig = filepath.Join(home, "."+DefaultConfigFile)

	configBody = []byte("theconfig")
)

func getHomeDir() string {
	home, _ := homedir.Dir()
	return home
}

func resetAppFsToMemFs(t *testing.T) {
	AppFs = afero.NewMemMapFs()
	require.NoError(t, AppFs.MkdirAll(homeConfigDir, 0o755))
	require.NoError(t, AppFs.MkdirAll(systemConfigDir, 0o755))
}

func Test_autodetectConfig(t *testing.T) {
	defer func() {
		AppFs = afero.NewOsFs()
	}()

	t.Run("test homeConfig returned if exists", func(t *testing.T) {
		resetAppFsToMemFs(t)
		require.NoError(t, afero.WriteFile(AppFs, homeConfig, configBody, 0o644))

		config := autodetectConfig()
		assert.Equal(t, homeConfig, config)
	})

	t.Run("test homeConfig returned even if systemConfig exists", func(t *testing.T) {
		resetAppFsToMemFs(t)
		require.NoError(t, afero.WriteFile(AppFs, homeConfig, configBody, 0o644))
		require.NoError(t, afero.WriteFile(AppFs, oldConfig, configBody, 0o644))
		require.NoError(t, afero.WriteFile(AppFs, systemConfig, configBody, 0o644))

		config := autodetectConfig()
		assert.Equal(t, homeConfig, config)
	})

	t.Run("test systemConfig returned", func(t *testing.T) {
		resetAppFsToMemFs(t)
		require.NoError(t, afero.WriteFile(AppFs, systemConfig, configBody, 0o644))

		config := autodetectConfig()
		assert.Equal(t, systemConfig, config)
	})
}

func Test_probeConfigMemFs(t *testing.T) {
	defer func() {
		AppFs = afero.NewOsFs()
	}()

	resetAppFsToMemFs(t)
	require.NoError(t, afero.WriteFile(AppFs, homeConfig, configBody, 0o644))

	assert.False(t, ProbeFile(homeConfigDir))
	assert.True(t, ProbeDir(homeConfigDir))
	assert.True(t, ProbeFile(homeConfig))
	assert.False(t, ProbeFile(homeConfig+".nothere"))
}

func Test_probeConfigOsFs(t *testing.T) {
	dir := t.TempDir()
	tmpConfig := filepath.Join(dir, DefaultConfigFile)
	require.NoError(t, afero.WriteFile(AppFs, tmpConfig, configBody, 0o000))

	assert.Equal(t, false, ProbeFile(tmpConfig))
}

func Test_inventoryPath(t *testing.T) {
	resetAppFsToMemFs(t)
	require.NoError(t, afero.WriteFile(AppFs, systemConfig, configBody, 0o644))
	require.NoError(t, afero.WriteFile(AppFs, systemInventory, []byte("---"), 0o644))

	path, ok := InventoryPath(systemConfig)
	assert.Equal(t, systemInventory, path)
	assert.True(t, ok)
}

func TestConfigParsing(t *testing.T) {
	t.Run("test config with space_mrn", func(t *testing.T) {
		data := `
agent_mrn: //agents.api.mondoo.app/spaces/musing-saha-952142/agents/1zDY7auR20SgrFfiGUT5qZWx6mE
api_endpoint: https://us.api.mondoo.com
api_proxy: http://192.168.4.4:3128
certificate: |
  -----BEGIN CERTIFICATE-----
  MIICV .. fis=
  -----END CERTIFICATE-----

mrn: //agents.api.mondoo.app/spaces/musing-saha-952142/serviceaccounts/1zDY7cJ7bA84JxxNBWDxBdui2xE
private_key: |
  -----BEGIN PRIVATE KEY-----
  MIG2AgE....C0Dvs=
  -----END PRIVATE KEY-----
space_mrn: //captain.api.mondoo.app/spaces/musing-saha-952142
`

		viper.SetConfigType("yaml")
		err := viper.ReadConfig(strings.NewReader(data))
		require.NoError(t, err)

		cfg, err := Read()
		require.NoError(t, err)
		assert.Equal(t, "//agents.api.mondoo.app/spaces/musing-saha-952142/agents/1zDY7auR20SgrFfiGUT5qZWx6mE", cfg.AgentMrn)
		assert.Equal(t, "//agents.api.mondoo.app/spaces/musing-saha-952142/serviceaccounts/1zDY7cJ7bA84JxxNBWDxBdui2xE", cfg.ServiceAccountMrn)
		assert.Equal(t, "-----BEGIN PRIVATE KEY-----\nMIG2AgE....C0Dvs=\n-----END PRIVATE KEY-----\n", cfg.PrivateKey)
		assert.Equal(t, "-----BEGIN CERTIFICATE-----\nMIICV .. fis=\n-----END CERTIFICATE-----\n", cfg.Certificate)
		assert.Equal(t, "//captain.api.mondoo.app/spaces/musing-saha-952142", cfg.GetScopeMrn())
		assert.Equal(t, "//captain.api.mondoo.app/spaces/musing-saha-952142", cfg.GetParentMrn())
	})

	t.Run("test space service account with scope_mrn", func(t *testing.T) {
		data := `
{
  "mrn": "//agents.api.mondoo.app/organizations/my-custom-org-id/serviceaccounts/2bB5gsCSGp2Tlwiyv7mKN9PRSHK",
  "certificate": "-----BEGIN CERTIFICATE-----\nMIICkjCCAhigAwI5MT...ju2MAkPg9dPc8MDZz7ukThmj1AZrap/5J166M=\n-----END CERTIFICATE-----\n",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIG2AgEAMBAGByqGSM...ju2MAkPg9dPc8MDZz7ukT/xQTS5FUmDNu7Rw8=\n-----END PRIVATE KEY-----\n",
  "scope_mrn": "//captain.api.mondoo.app/organizations/my-custom-org-id",
  "api_endpoint": "https://us.api.mondoo.com",
  "space_mrn": "//captain.api.mondoo.app/organizations/my-custom-org-id",
  "typename": "ServiceAccountCredential"
}
`
		viper.SetConfigType("yaml")
		err := viper.ReadConfig(strings.NewReader(data))
		require.NoError(t, err)

		cfg, err := Read()
		require.NoError(t, err)
		assert.Equal(t, "", cfg.AgentMrn)
		assert.Equal(t, "//agents.api.mondoo.app/organizations/my-custom-org-id/serviceaccounts/2bB5gsCSGp2Tlwiyv7mKN9PRSHK", cfg.ServiceAccountMrn)
		assert.Equal(t, "-----BEGIN PRIVATE KEY-----\nMIG2AgEAMBAGByqGSM...ju2MAkPg9dPc8MDZz7ukT/xQTS5FUmDNu7Rw8=\n-----END PRIVATE KEY-----\n", cfg.PrivateKey)
		assert.Equal(t, "-----BEGIN CERTIFICATE-----\nMIICkjCCAhigAwI5MT...ju2MAkPg9dPc8MDZz7ukThmj1AZrap/5J166M=\n-----END CERTIFICATE-----\n", cfg.Certificate)
		assert.Equal(t, "//captain.api.mondoo.app/organizations/my-custom-org-id", cfg.GetScopeMrn())
		assert.Equal(t, "//captain.api.mondoo.app/organizations/my-custom-org-id", cfg.GetParentMrn())
	})

	t.Run("test org service account with scope_mrn", func(t *testing.T) {
		data := `
{
  "mrn": "//agents.api.mondoo.app/spaces/my-space-id/serviceaccounts/2bUj407V4GF4IKxg3Qn6NhWCr6x",
  "certificate": "-----BEGIN CERTIFICATE-----\nMIICfDCCAgKgAwIBAgIQGwVGMqyjkNaCGTA96p/...\n2mm3zQE7mUokDf4qY3+SDw==\n-----END CERTIFICATE-----\n",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIG2AgEAMBAGByqGSM49AgEGBSuBBAAiBIGeMIG...ILipt7Y8zEZ7PRQPkGUYpWDE8=\n-----END PRIVATE KEY-----\n",
  "scope_mrn": "//captain.api.mondoo.app/spaces/my-space-id",
  "api_endpoint": "https://us.api.mondoo.com",
  "space_mrn": "//captain.api.mondoo.app/spaces/my-space-id",
  "typename": "ServiceAccountCredential"
}
`

		viper.SetConfigType("yaml")
		err := viper.ReadConfig(strings.NewReader(data))
		require.NoError(t, err)

		cfg, err := Read()
		require.NoError(t, err)
		assert.Equal(t, "", cfg.AgentMrn)
		assert.Equal(t, "//agents.api.mondoo.app/spaces/my-space-id/serviceaccounts/2bUj407V4GF4IKxg3Qn6NhWCr6x", cfg.ServiceAccountMrn)
		assert.Equal(t, "-----BEGIN PRIVATE KEY-----\nMIG2AgEAMBAGByqGSM49AgEGBSuBBAAiBIGeMIG...ILipt7Y8zEZ7PRQPkGUYpWDE8=\n-----END PRIVATE KEY-----\n", cfg.PrivateKey)
		assert.Equal(t, "-----BEGIN CERTIFICATE-----\nMIICfDCCAgKgAwIBAgIQGwVGMqyjkNaCGTA96p/...\n2mm3zQE7mUokDf4qY3+SDw==\n-----END CERTIFICATE-----\n", cfg.Certificate)
		assert.Equal(t, "//captain.api.mondoo.app/spaces/my-space-id", cfg.GetScopeMrn())
		assert.Equal(t, "//captain.api.mondoo.app/spaces/my-space-id", cfg.GetParentMrn())
	})

	t.Run("test WIF config format", func(t *testing.T) {
		data := `
{
    "universeDomain": "https://api.example.com",
    "scopes": [
        "//iam.api.mondoo.app/roles/agent"
    ],
    "type": "external_account",
    "audience": "//captain.api.mondoo.app/spaces/test-space-id",
    "subjectTokenType": "urn:ietf:params:oauth:token-type:jwt",
    "issuerUri": "https://accounts.google.com"
}
`
		viper.SetConfigType("json")
		err := viper.ReadConfig(strings.NewReader(data))
		require.NoError(t, err)

		if viper.GetString("type") == "external_account" {
			viper.Set("auth", map[string]string{"method": "wif"})

			if audience := viper.GetString("audience"); audience != "" {
				viper.Set("audience", audience)
			}

			if issuerUri := viper.GetString("issuerUri"); issuerUri != "" {
				viper.Set("issuer_uri", issuerUri)
			}

			if universeDomain := viper.GetString("universeDomain"); universeDomain != "" {
				viper.Set("api_endpoint", universeDomain)
			}
		}

		cfg, err := Read()
		require.NoError(t, err)

		assert.Equal(t, "wif", cfg.Authentication.Method)
		assert.Equal(t, "//captain.api.mondoo.app/spaces/test-space-id", cfg.Audience)
		assert.Equal(t, "https://accounts.google.com", cfg.IssuerURI)
		assert.Equal(t, "https://api.example.com", cfg.APIEndpoint)
	})
}

func TestIsWifConfigFormat(t *testing.T) {
	resetAppFsToMemFs(t)

	t.Run("valid WIF config", func(t *testing.T) {
		wifConfig := `{
			"universeDomain": "https://api.example.com",
			"scopes": ["//iam.api.mondoo.app/roles/agent"],
			"type": "external_account",
			"audience": "//captain.api.mondoo.app/spaces/test-space-id",
			"subjectTokenType": "urn:ietf:params:oauth:token-type:jwt",
			"issuerUri": "https://accounts.google.com"
		}`
		configPath := filepath.Join(homeConfigDir, "wif_config.json")
		require.NoError(t, afero.WriteFile(AppFs, configPath, []byte(wifConfig), 0o644))

		isWif, err := IsWifConfigFormat(configPath)
		require.NoError(t, err)
		assert.True(t, isWif)
	})

	t.Run("invalid JSON file", func(t *testing.T) {
		invalidJSON := `{ "this is not valid JSON`
		configPath := filepath.Join(homeConfigDir, "invalid.json")
		require.NoError(t, afero.WriteFile(AppFs, configPath, []byte(invalidJSON), 0o644))

		isWif, err := IsWifConfigFormat(configPath)
		require.NoError(t, err)
		assert.False(t, isWif)
	})

	t.Run("non-WIF JSON config", func(t *testing.T) {
		nonWifConfig := `{
			"api_endpoint": "https://us.api.mondoo.com",
			"mrn": "//agents.api.mondoo.app/spaces/my-space-id/serviceaccounts/abc123",
			"private_key": "-----BEGIN PRIVATE KEY-----\\nkey content\\n-----END PRIVATE KEY-----\\n"
		}`
		configPath := filepath.Join(homeConfigDir, "non_wif.json")
		require.NoError(t, afero.WriteFile(AppFs, configPath, []byte(nonWifConfig), 0o644))

		isWif, err := IsWifConfigFormat(configPath)
		require.NoError(t, err)
		assert.False(t, isWif)
	})

	t.Run("WIF config missing required fields", func(t *testing.T) {
		missingFields := `{
			"type": "external_account",
			"subjectTokenType": "urn:ietf:params:oauth:token-type:jwt"
		}`
		configPath := filepath.Join(homeConfigDir, "missing_fields.json")
		require.NoError(t, afero.WriteFile(AppFs, configPath, []byte(missingFields), 0o644))

		isWif, err := IsWifConfigFormat(configPath)
		require.NoError(t, err)
		assert.False(t, isWif)
	})
}

func TestConvertWifConfig(t *testing.T) {
	resetAppFsToMemFs(t)

	t.Run("successful conversion", func(t *testing.T) {
		wifConfig := `{
			"universeDomain": "https://api.example.com",
			"scopes": ["//iam.api.mondoo.app/roles/agent"],
			"type": "external_account",
			"audience": "//captain.api.mondoo.app/spaces/test-space-id",
			"subjectTokenType": "urn:ietf:params:oauth:token-type:jwt",
			"issuerUri": "https://accounts.google.com"
		}`
		configPath := filepath.Join(homeConfigDir, "wif_config.json")
		require.NoError(t, afero.WriteFile(AppFs, configPath, []byte(wifConfig), 0o644))

		v := viper.New()
		err := ConvertWifConfig(configPath, v)
		require.NoError(t, err)

		assert.Equal(t, "wif", v.GetStringMapString("auth")["method"])
		assert.Equal(t, "//captain.api.mondoo.app/spaces/test-space-id", v.GetString("audience"))
		assert.Equal(t, "https://accounts.google.com", v.GetString("issuer_uri"))
		assert.Equal(t, "https://api.example.com", v.GetString("api_endpoint"))
		assert.Equal(t, "https://api.example.com", v.GetString("universeDomain"))
		assert.Equal(t, "external_account", v.GetString("type"))
	})

	t.Run("missing universeDomain", func(t *testing.T) {
		missingDomain := `{
			"type": "external_account",
			"audience": "//captain.api.mondoo.app/spaces/test-space-id",
			"subjectTokenType": "urn:ietf:params:oauth:token-type:jwt",
			"issuerUri": "https://accounts.google.com"
		}`
		configPath := filepath.Join(homeConfigDir, "missing_domain.json")
		require.NoError(t, afero.WriteFile(AppFs, configPath, []byte(missingDomain), 0o644))

		v := viper.New()
		err := ConvertWifConfig(configPath, v)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required 'universeDomain' field")
	})

	t.Run("missing audience", func(t *testing.T) {
		missingAudience := `{
			"universeDomain": "https://api.example.com",
			"type": "external_account",
			"subjectTokenType": "urn:ietf:params:oauth:token-type:jwt",
			"issuerUri": "https://accounts.google.com"
		}`
		configPath := filepath.Join(homeConfigDir, "missing_audience.json")
		require.NoError(t, afero.WriteFile(AppFs, configPath, []byte(missingAudience), 0o644))

		v := viper.New()
		err := ConvertWifConfig(configPath, v)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required 'audience' field")
	})
}
