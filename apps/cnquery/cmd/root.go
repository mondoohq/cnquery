package cmd

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/sysinfo"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/plugins/scope"
)

const (
	askForPasswordValue = ">passwordisnotset<"
	rootCmdDesc         = "cnquery is a cloud-native tool for querying your entire fleet\n"

	// we send a 78 exit code to prevent systemd service from restart
	ConfigurationErrorCode = 78
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cnquery",
	Short: "cnquery CLI",
	Long:  theme.DefaultTheme.Landing + "\n\n" + rootCmdDesc,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogger(cmd)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// normal cli handling
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// NOTE: we need to call this super early, otherwise the CLI color output on Windows is broken for the first lines
	// since the log instance is already initialized, replace default zerolog color output with our own
	// use color logger by default
	logger.CliCompactLogger(logger.LogOutputWriter)
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	config.DefaultConfigFile = "mondoo.yml"

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("log-level", "info", "Set log level: error, warn, info, debug, trace")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindEnv("features")

	config.Init(rootCmd)
}

func initLogger(cmd *cobra.Command) {
	// environment variables always over-write custom flags
	envLevel, ok := logger.GetEnvLogLevel()
	if ok {
		logger.Set(envLevel)
		return
	}

	// retrieve log-level from flags
	level := viper.GetString("log-level")
	if v := viper.GetBool("verbose"); v {
		level = "debug"
	}
	logger.Set(level)
}

// storeRecording stores tracked commands and files into the recording file
func storeRecording(m *motor.Motor) {
	if m.IsRecording() {
		filename := viper.GetString("record-file")
		if filename == "" {
			filename = "recording-" + time.Now().Format("20060102150405") + ".toml"
		}
		log.Info().Str("filename", filename).Msg("store recordings")
		data := m.Recording()
		os.WriteFile(filename, data, 0o700)
	}
}

func filterAssetByPlatformID(assetList []*asset.Asset, selectionID string) (*asset.Asset, error) {
	var foundAsset *asset.Asset
	for i := range assetList {
		assetObj := assetList[i]
		for j := range assetObj.PlatformIds {
			if assetObj.PlatformIds[j] == selectionID {
				return assetObj, nil
			}
		}
	}

	if foundAsset == nil {
		return nil, errors.New("could not find an asset with the provided identifer: " + selectionID)
	}
	return foundAsset, nil
}

func defaultRangerPlugins(sysInfo *sysinfo.SystemInfo, features cnquery.Features) []ranger.ClientPlugin {
	plugins := []ranger.ClientPlugin{}
	plugins = append(plugins, scope.NewRequestIDRangerPlugin())
	plugins = append(plugins, sysInfoHeader(sysInfo, features))
	return plugins
}

func sysInfoHeader(sysInfo *sysinfo.SystemInfo, features cnquery.Features) ranger.ClientPlugin {
	const (
		HttpHeaderUserAgent      = "User-Client"
		HttpHeaderClientFeatures = "Mondoo-Features"
		HttpHeaderPlatformID     = "Mondoo-PlatformID"
	)

	h := http.Header{}
	h.Set(HttpHeaderUserAgent, scope.XInfoHeader(map[string]string{
		"cnquery": cnquery.Version,
		"build":   cnquery.Build,
		"PN":      sysInfo.Platform.Name,
		"PR":      sysInfo.Platform.Version,
		"PA":      sysInfo.Platform.Arch,
		"IP":      sysInfo.IP,
		"HN":      sysInfo.Hostname,
	}))
	h.Set(HttpHeaderClientFeatures, features.Encode())
	h.Set(HttpHeaderPlatformID, sysInfo.PlatformId)
	return scope.NewCustomHeaderRangerPlugin(h)
}

func GenerateMarkdown(dir string) error {
	rootCmd.DisableAutoGenTag = true

	// We need to remove our fancy logo from the markdown output,
	// since it messes with the formatting.
	rootCmd.Long = rootCmdDesc
	return doc.GenMarkdownTree(rootCmd, dir)
}
