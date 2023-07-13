package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/cockroachdb/errors"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

var (
	UserProvidedPath     string
	Path                 string
	LoadedConfig         bool
	DefaultConfigFile    = "mondoo.yml"
	DefaultInventoryFile = "inventory.yml"
	Source               string
	AppFs                afero.Fs
)

func init() {
	AppFs = afero.NewOsFs()
}

func probePath(path string, asFile bool) bool {
	stat, err := AppFs.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Warn().Str("path", path).Msg("detected Schrödinger's path, cannot detect if it is usable")
		return false
	}

	if !asFile {
		return stat.Mode().IsDir()
	}

	if !stat.Mode().IsRegular() {
		log.Warn().Str("path", path).Msg("cannot use configuration file, it doesn't look like a regular file")
		return false
	}

	f, err := AppFs.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	return true
}

// ProbeDir tests a path if it's a directory and it exists
func ProbeDir(path string) bool {
	return probePath(path, false)
}

// ProbeFile tests a path if it's a file and if we can access it
func ProbeFile(path string) bool {
	return probePath(path, true)
}

// HomePath returns the user-level path for Mondoo. The given argument
// is appended and checked if it is accessible and regular.
// Returns error if the home directory could not be determined.
func HomePath(childPath ...string) (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", errors.Wrap(err, "failed to determine user home directory")
	}

	parts := append([]string{home, ".config", "mondoo"}, childPath...)
	homeConfig := filepath.Join(parts...)
	return homeConfig, nil
}

func systemPath(isConfig bool, childPath ...string) string {
	var parts []string
	if runtime.GOOS == "windows" {
		parts = append([]string{`C:\ProgramData\Mondoo\`}, childPath...)
	} else if isConfig {
		parts = append([]string{"/etc", "opt", "mondoo"}, childPath...)
	} else {
		parts = append([]string{"/opt", "mondoo"}, childPath...)
	}

	systemConfig := filepath.Join(parts...)
	return systemConfig
}

// SystemConfigPath returns the system-level config path for Mondoo. The given argument
// is appended and checked if it is accessible and regular.
func SystemConfigPath(childPath ...string) string {
	return systemPath(true, childPath...)
}

// SystemDataPath returns the system-level data path for Mondoo. The given argument
// is appended and checked if it is accessible and regular.
func SystemDataPath(childPath ...string) string {
	return systemPath(false, childPath...)
}

func autodetectConfig() string {
	homeConfig, err := HomePath(DefaultConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to autodetect mondoo config")
	}
	if ProbeFile(homeConfig) {
		return homeConfig
	}

	sysConfig := SystemConfigPath(DefaultConfigFile)
	if ProbeFile(sysConfig) {
		return sysConfig
	}

	// Note: At this point we don't have any config. However, we will have to
	// set up a potential config, which may be auto-created for us. Pointing it to
	// the system config by default may be problematic, since:
	// 1. if you're using a regular user, you can't create/write that path
	// 2. if you're root, we probably don't want to auto-create it there
	//    due to the far-reaching impact as it may influence all users

	return homeConfig
}

// returns the inventory path relative to the config file
func InventoryPath(configPath string) (string, bool) {
	inventoryPath := filepath.Join(filepath.Dir(configPath), DefaultInventoryFile)
	return inventoryPath, ProbeFile(inventoryPath)
}
