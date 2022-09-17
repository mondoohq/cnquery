package config

import (
	"path/filepath"
	"testing"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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

	assert.Equal(t, false, probeConfig(homeConfigDir))
	assert.Equal(t, true, probeConfig(homeConfig))
	assert.Equal(t, false, probeConfig(homeConfig+".nothere"))
}

func Test_probeConfigOsFs(t *testing.T) {
	dir := t.TempDir()
	tmpConfig := filepath.Join(dir, DefaultConfigFile)
	afero.WriteFile(AppFs, tmpConfig, configBody, 0o000)

	assert.Equal(t, false, probeConfig(tmpConfig))
}

func Test_inventoryPath(t *testing.T) {
	resetAppFsToMemFs()
	afero.WriteFile(AppFs, systemConfig, configBody, 0o644)
	afero.WriteFile(AppFs, systemInventory, []byte("---"), 0o644)

	path, ok := InventoryPath(systemConfig)
	assert.Equal(t, systemInventory, path)
	assert.True(t, ok)
}
