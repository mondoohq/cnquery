package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/logger"
)

/*
	Configuration is loaded in this order:
	ENV -> ~/.mondoo.conf -> defaults
*/

// Path is the currently loaded config location
// or default if no config exits
var (
	UserProvidedPath     string
	Path                 string
	LoadedConfig         bool
	DefaultConfigFile    = "mondoo.yml"
	DefaultInventoryFile = "inventory.yml"
	Source               string
	AppFs                afero.Fs
	Features             cnquery.Features
)

// Init initializes and loads the mondoo config
func Init(rootCmd *cobra.Command) {
	cobra.OnInitialize(initConfig)
	AppFs = afero.NewOsFs()
	Features = getFeatures()
	// persistent flags are global for the application
	rootCmd.PersistentFlags().StringVar(&UserProvidedPath, "config", "", "Set config file path (default is $HOME/.config/mondoo/mondoo.yml)")
}

func getFeatures() cnquery.Features {
	bitSet := make([]bool, 256)
	flags := []byte{}

	for _, f := range cnquery.DefaultFeatures {
		if !bitSet[f] {
			bitSet[f] = true
			flags = append(flags, f)
		}
	}

	envFeatures := viper.GetStringSlice("features")
	for _, name := range envFeatures {
		flag, ok := cnquery.FeaturesValue[name]
		if ok {
			if !bitSet[byte(flag)] {
				bitSet[byte(flag)] = true
				flags = append(flags, byte(flag))
			}
		} else {
			log.Warn().Str("feature", name).Msg("could not parse feature")
		}
	}

	return cnquery.Features(flags)
}

func isAccessible(path string) bool {
	f, err := AppFs.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	return true
}

// Test if we can use a config file in a path. Returns true if that's the case
func probeConfig(path string) bool {
	if stat, err := AppFs.Stat(path); err == nil {
		if !stat.Mode().IsRegular() {
			log.Warn().Str("path", path).Msg("cannot use configuration file, it doesn't look like a regular file")
			return false
		}
		return isAccessible(path)
	} else if os.IsNotExist(err) {
		return false
	} else {
		log.Warn().Str("path", path).Msg("detected Schrödinger's config file, cannot detect if it is usable")
	}

	return false
}

// HomePath returns the user-level configuration for Mondoo.
func HomePath() (string, bool, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", false, errors.Wrap(err, "failed to determine user home directory")
	}

	homeConfig := filepath.Join(home, ".config", "mondoo", DefaultConfigFile)
	useHome := probeConfig(homeConfig)

	return homeConfig, useHome, nil
}

func SystemPath() (string, bool) {
	var systemConfig string
	if runtime.GOOS == "windows" {
		systemConfig = filepath.Join(`C:\ProgramData\Mondoo\`, DefaultConfigFile)
	} else {
		systemConfig = filepath.Join("/etc", "opt", "mondoo", DefaultConfigFile)
	}
	return systemConfig, probeConfig(systemConfig)
}

func autodetectConfig() string {
	homeConfig, exists, err := HomePath()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to autodetect mondoo config")
	}
	if exists {
		return homeConfig
	}

	if path, ok := SystemPath(); ok {
		return path
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
	return inventoryPath, probeConfig(inventoryPath)
}

func initConfig() {
	viper.SetConfigType("yaml")

	Path = strings.TrimSpace(UserProvidedPath)

	// fallback to env variable if provided, but only if --config is not used and no loc
	if len(Path) == 0 && len(os.Getenv("MONDOO_CONFIG_PATH")) > 0 {
		Source = "$MONDOO_CONFIG_PATH"
		Path = os.Getenv("MONDOO_CONFIG_PATH")
	} else if len(Path) != 0 {
		Source = "--config"
	} else {
		Source = "default"
	}

	// check if the default config file is available
	if Path == "" {
		Path = autodetectConfig()
	}

	// we set this here, so that sub commands that rely on writing config, can use the default config
	viper.SetConfigFile(Path)

	// if the file exists, load it
	_, err := AppFs.Stat(Path)
	if err == nil {
		log.Debug().Str("configfile", viper.ConfigFileUsed()).Msg("try to load local config file")
		if err := viper.ReadInConfig(); err == nil {
			LoadedConfig = true
		} else {
			LoadedConfig = false
			log.Error().Err(err).Str("path", Path).Msg("could not read config file")
		}
	}

	// by default it uses console output, for production we may want to set it to json output
	if viper.GetString("log.format") == "json" {
		logger.UseJSONLogging(logger.LogOutputWriter)
	}

	if viper.GetBool("log.color") == true {
		logger.CliCompactLogger(logger.LogOutputWriter)
	}

	// override values with env variables
	viper.SetEnvPrefix("mondoo")
	// to parse env variables properly we need to replace some chars
	// all hypens need to be underscores
	// all dots neeto to be underscores
	replacer := strings.NewReplacer("-", "_", ".", "_")
	viper.SetEnvKeyReplacer(replacer)

	// read in environment variables that match
	viper.AutomaticEnv()
}

func DisplayUsedConfig() {
	// print config file
	if !LoadedConfig && len(UserProvidedPath) > 0 {
		log.Warn().Msg("could not load configuration file " + UserProvidedPath)
	} else if LoadedConfig {
		log.Info().Msg("loaded configuration from " + viper.ConfigFileUsed() + " using source " + Source)
	} else {
		log.Info().Msg("no configuration file provided")
	}
}
