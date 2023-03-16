package rangerclient

import (
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	cnquery_config "go.mondoo.com/cnquery/apps/cnquery/cmd/config"
	"go.mondoo.com/ranger-rpc"
)

type RangerClientOpts struct {
	Proxy string
}

// NewRangerClient will set up the underlyig ranger client
// with the appropriate proxy if needed.
func NewRangerClient() (*http.Client, error) {
	proxy := getMondooAPIProxy()

	rangerClient, err := ranger.HttpClient(&ranger.HttpClientOpts{
		Proxy: proxy,
	})
	if err != nil {
		return nil, err
	}

	return rangerClient, nil
}

// getMondooAPIProxy will in order of precedence use the proxy info found in
// 1) MONDO_API_PROXY env var
// 2) the --api-proxy CLI parameter
// 3) the api_proxy setting in the config file
func getMondooAPIProxy() string {
	proxy, envSet := os.LookupEnv("MONDOO_API_PROXY")
	if envSet {
		return proxy
	}

	proxy = viper.GetString("api-proxy")
	if proxy != "" {
		return proxy
	}

	opts, optsErr := cnquery_config.ReadConfig()
	if optsErr != nil {
		log.Fatal().Err(optsErr).Msg("could not load configuration")
	}

	return opts.APIProxy
}
