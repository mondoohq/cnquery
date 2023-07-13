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

func resetAppFsToMemFs() {
	AppFs = afero.NewMemMapFs()
	AppFs.MkdirAll(homeConfigDir, 0o755)
	AppFs.MkdirAll(systemConfigDir, 0o755)
}

func Test_autodetectConfig(t *testing.T) {
	defer func() {
		AppFs = afero.NewOsFs()
	}()

	t.Run("test homeConfig returned if exists", func(t *testing.T) {
		resetAppFsToMemFs()
		afero.WriteFile(AppFs, homeConfig, configBody, 0o644)

		config := autodetectConfig()
		assert.Equal(t, homeConfig, config)
	})

	t.Run("test homeConfig returned even if systemConfig exists", func(t *testing.T) {
		resetAppFsToMemFs()
		afero.WriteFile(AppFs, homeConfig, configBody, 0o644)
		afero.WriteFile(AppFs, oldConfig, configBody, 0o644)
		afero.WriteFile(AppFs, systemConfig, configBody, 0o644)

		config := autodetectConfig()
		assert.Equal(t, homeConfig, config)
	})

	t.Run("test systemConfig returned", func(t *testing.T) {
		resetAppFsToMemFs()
		afero.WriteFile(AppFs, systemConfig, configBody, 0o644)

		config := autodetectConfig()
		assert.Equal(t, systemConfig, config)
	})
}

func Test_probeConfigMemFs(t *testing.T) {
	defer func() {
		AppFs = afero.NewOsFs()
	}()

	resetAppFsToMemFs()
	afero.WriteFile(AppFs, homeConfig, configBody, 0o644)

	assert.False(t, ProbeFile(homeConfigDir))
	assert.True(t, ProbeDir(homeConfigDir))
	assert.True(t, ProbeFile(homeConfig))
	assert.False(t, ProbeFile(homeConfig+".nothere"))
}

func Test_probeConfigOsFs(t *testing.T) {
	dir := t.TempDir()
	tmpConfig := filepath.Join(dir, DefaultConfigFile)
	afero.WriteFile(AppFs, tmpConfig, configBody, 0o000)

	assert.Equal(t, false, ProbeFile(tmpConfig))
}

func Test_inventoryPath(t *testing.T) {
	resetAppFsToMemFs()
	afero.WriteFile(AppFs, systemConfig, configBody, 0o644)
	afero.WriteFile(AppFs, systemInventory, []byte("---"), 0o644)

	path, ok := InventoryPath(systemConfig)
	assert.Equal(t, systemInventory, path)
	assert.True(t, ok)
}

func TestConfigParsing(t *testing.T) {
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
	viper.ReadConfig(strings.NewReader(data))

	cfg, err := Read()
	require.NoError(t, err)
	assert.Equal(t, "//agents.api.mondoo.app/spaces/musing-saha-952142/agents/1zDY7auR20SgrFfiGUT5qZWx6mE", cfg.AgentMrn)
	assert.Equal(t, "//agents.api.mondoo.app/spaces/musing-saha-952142/serviceaccounts/1zDY7cJ7bA84JxxNBWDxBdui2xE", cfg.ServiceAccountMrn)
	assert.Equal(t, "-----BEGIN PRIVATE KEY-----\nMIG2AgE....C0Dvs=\n-----END PRIVATE KEY-----\n", cfg.PrivateKey)
	assert.Equal(t, "-----BEGIN CERTIFICATE-----\nMIICV .. fis=\n-----END CERTIFICATE-----\n", cfg.Certificate)
	assert.Equal(t, "http://192.168.4.4:3128", cfg.APIProxy)
}
