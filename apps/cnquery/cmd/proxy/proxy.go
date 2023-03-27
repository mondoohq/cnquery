package proxy

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	cnquery_config "go.mondoo.com/cnquery/apps/cnquery/cmd/config"
)

var activeProxy string

func GetAPIProxy() string {
	return activeProxy
}

func ProxyInit() {
	proxy, envSet := os.LookupEnv("MONDOO_API_PROXY")
	if envSet {
		activeProxy = proxy
		return
	}

	proxy = viper.GetString("api-proxy")
	if proxy != "" {
		activeProxy = proxy
		return
	}

	cliOpts, err := cnquery_config.ReadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("error while reading in mondoo config file")
	}
	activeProxy = cliOpts.APIProxy
	return
}
