// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
)

/*
	Configuration is loaded in this order:
	ENV -> ~/.mondoo.conf -> defaults
*/

// Path is the currently loaded config location
// or default if no config exits
var (
	Features cnquery.Features
)

const (
	configSourceBase64 = "$MONDOO_CONFIG_BASE64"
	defaultAPIendpoint = "https://us.api.mondoo.com"
)

// Init initializes and loads the mondoo config
func Init(rootCmd *cobra.Command) {
	cobra.OnInitialize(InitViperConfig, func() {
		Features = getFeatures()
	})
	// persistent flags are global for the application
	rootCmd.PersistentFlags().StringVar(&UserProvidedPath, "config", "", "Set config file path (default $HOME/.config/mondoo/mondoo.yml)")
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

func InitViperConfig() {
	viper.SetConfigType("yaml")

	Path = strings.TrimSpace(UserProvidedPath)
	// base 64 config env setting has always precedence
	if len(os.Getenv("MONDOO_CONFIG_BASE64")) > 0 {
		Source = configSourceBase64
		decodedData, err := base64.StdEncoding.DecodeString(os.Getenv("MONDOO_CONFIG_BASE64"))
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse base64 ")
		}
		err = viper.ReadConfig(bytes.NewBuffer(decodedData))
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
	} else if len(Path) == 0 && len(os.Getenv("MONDOO_CONFIG_PATH")) > 0 {
		// fallback to env variable if provided, but only if --config is not used
		Source = "$MONDOO_CONFIG_PATH"
		Path = os.Getenv("MONDOO_CONFIG_PATH")
	} else if len(Path) != 0 {
		Source = "--config"
	} else {
		Source = "default"
	}
	if strings.HasPrefix(Path, AWS_SSM_PARAMETERSTORE_PREFIX) {
		err := loadAwsSSMParameterStore(Path)
		if err != nil {
			LoadedConfig = false
			log.Error().Err(err).Str("path", Path).Msg("could not load aws parameter store config")
		} else {
			LoadedConfig = true
		}
	}

	// check if the default config file is available
	if Path == "" && Source != configSourceBase64 {
		Path = autodetectConfig()
	}

	if Source != configSourceBase64 {
		// we set this here, so that sub commands that rely on writing config, can use the default config
		viper.SetConfigFile(Path)

		// if the file exists, load it
		_, err := AppFs.Stat(Path)
		if err == nil {
			log.Debug().Str("configfile", viper.ConfigFileUsed()).Msg("try to load local config file")

			// Check if this is a WIF config file
			isWif, wifErr := IsWifConfigFormat(Path)
			if wifErr != nil {
				log.Debug().Err(wifErr).Msg("error checking for WIF config format")
			} else if isWif {
				log.Debug().Msg("detected WIF config file format")

				// Convert the WIF config to Viper format
				if err := ConvertWifConfig(Path, viper.GetViper()); err != nil {
					LoadedConfig = false
					log.Error().Err(err).Str("path", Path).Msg("could not convert WIF config file")
				} else {
					LoadedConfig = true
					log.Debug().Msg("successfully converted WIF config")
				}
			} else {
				// Regular config file - load it normally
				if err := viper.ReadInConfig(); err == nil {
					LoadedConfig = true
				} else {
					LoadedConfig = false
					log.Error().Err(err).Str("path", Path).Msg("could not read config file")
				}
			}
		}
	}

	// Check if this is a WIF config file
	// This detects the new format: {"type": "external_account", "audience": "...", "issuerUri": "..."}
	if viper.GetString("type") == "external_account" {
		log.Debug().Msg("detected WIF config format")

		// Configure authentication method
		if !viper.IsSet("auth") {
			viper.Set("auth", map[string]string{"method": "wif"})
		} else {
			// If auth exists but method isn't set, set it to wif
			authMap := viper.GetStringMap("auth")
			if _, exists := authMap["method"]; !exists {
				viper.Set("auth.method", "wif")
			}
		}

		// Set the audience if available
		if audience := viper.GetString("audience"); audience != "" {
			viper.Set("audience", audience)
		}

		// Set the issuer URI if available
		if issuerUri := viper.GetString("issuerUri"); issuerUri != "" {
			viper.Set("issuer_uri", issuerUri)
		}

		// Set the API endpoint from universeDomain if available
		if universeDomain := viper.GetString("universeDomain"); universeDomain != "" {
			viper.Set("api_endpoint", universeDomain)
		}

		// Log the detected configuration
		log.Debug().
			Str("audience", viper.GetString("audience")).
			Str("issuerUri", viper.GetString("issuer_uri")).
			Str("apiEndpoint", viper.GetString("api_endpoint")).
			Msg("configured WIF authentication from config file")
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
	// all hyphens need to be underscores
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
	} else if Source == configSourceBase64 {
		log.Info().Msg("loaded configuration from environment using source " + Source)
	} else {
		log.Info().Msg("no Mondoo configuration file provided, using defaults")
	}
}

func Read() (*Config, error) {
	// load viper config into a struct
	var opts Config
	err := viper.Unmarshal(&opts)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode into config struct")
	}

	return &opts, nil
}

type Config struct {
	// inherit common config
	CommonOpts `mapstructure:",squash"`

	// Asset Category
	Category               string `json:"category,omitempty" mapstructure:"category"`
	AutoDetectCICDCategory bool   `json:"detect-cicd,omitempty" mapstructure:"detect-cicd"`
}

type CommonOpts struct {
	// client identifier
	AgentMrn string `json:"agent_mrn,omitempty" mapstructure:"agent_mrn"`

	// service account credentials
	ServiceAccountMrn string `json:"mrn,omitempty" mapstructure:"mrn"`
	// The scope mrn is used to scope the service account to a specific organization or space.
	ScopeMrn string `json:"scope_mrn,omitempty" mapstructure:"scope_mrn"`
	// Deprecated: use scope_mrn instead
	ParentMrn string `json:"parent_mrn,omitempty" mapstructure:"parent_mrn"`
	// Deprecated: use scope_mrn instead
	SpaceMrn    string `json:"space_mrn,omitempty" mapstructure:"space_mrn"`
	PrivateKey  string `json:"private_key,omitempty" mapstructure:"private_key"`
	Certificate string `json:"certificate,omitempty" mapstructure:"certificate"`
	APIEndpoint string `json:"api_endpoint,omitempty" mapstructure:"api_endpoint"`

	// authentication
	Authentication *CliConfigAuthentication `json:"auth,omitempty" mapstructure:"auth"`

	// Workload Identity Federation fields
	Audience  string `json:"audience,omitempty" mapstructure:"audience"`
	IssuerURI string `json:"issuer_uri,omitempty" mapstructure:"issuer_uri"`

	// client features
	Features []string `json:"features,omitempty" mapstructure:"features"`

	// API Proxy for communicating with Mondoo Platform API
	APIProxy string `json:"api_proxy,omitempty" mapstructure:"api_proxy"`

	// labels that will be applied to all assets
	Labels map[string]string `json:"labels,omitempty" mapstructure:"labels"`

	// annotations that will be applied to all assets
	Annotations map[string]string `json:"annotations,omitempty" mapstructure:"annotations"`
}

type CliConfigAuthentication struct {
	Method string `json:"method,omitempty" mapstructure:"method"`
}

func (c *CommonOpts) GetFeatures() cnquery.Features {
	bitSet := make([]bool, 256)
	flags := []byte{}

	for _, f := range cnquery.DefaultFeatures {
		if !bitSet[f] {
			bitSet[f] = true
			flags = append(flags, f)
		}
	}

	for _, name := range c.Features {
		flag, ok := cnquery.FeaturesValue[name]
		if ok {
			if !bitSet[byte(flag)] {
				bitSet[byte(flag)] = true
				flags = append(flags, byte(flag))
			}
		} else {
			log.Warn().Str("feature", name).Msg("could not get a feature")
		}
	}

	return flags
}

// GetServiceCredential returns the service credential that is defined in the config.
// If no service credential is defined, it will return nil.
func (c *CommonOpts) GetServiceCredential() *upstream.ServiceAccountCredentials {
	// If we have an authentication method defined, use it
	if c.Authentication != nil {
		switch c.Authentication.Method {
		case "ssh":
			log.Info().Msg("using ssh authentication method, generate temporary credentials")
			serviceAccount, err := upstream.ExchangeSSHKey(c.UpstreamApiEndpoint(), c.ServiceAccountMrn, c.GetParentMrn())
			if err != nil {
				log.Error().Err(err).Msg("could not exchange ssh key")
				return nil
			}
			return serviceAccount
		case "wif":
			log.Info().Msg("using wif authentication method, generate temporary credentials")

			// Validate required fields for WIF
			if c.Audience == "" {
				log.Error().Msg("missing required field 'audience' for WIF authentication")
				return nil
			}

			// If apiEndpoint is not set, check for universeDomain in viper
			if c.APIEndpoint == "" {
				universeDomain := viper.GetString("universeDomain")
				if universeDomain != "" {
					c.APIEndpoint = universeDomain
					log.Debug().Str("apiEndpoint", c.APIEndpoint).Msg("using universeDomain as API endpoint for WIF authentication")
				}
			}

			// Now check if we have an API endpoint
			if c.APIEndpoint == "" {
				log.Error().Msg("missing required API endpoint for WIF authentication. Set 'universeDomain' in config")
				return nil
			}

			// Provide default issuer if not set
			issuerURI := c.IssuerURI
			if issuerURI == "" {
				log.Error().Msg("missing required issuer URI for WIF authentication. Set 'issuer_uri' in config ")
				return nil
			}

			serviceAccount, err := upstream.ExchangeExternalToken(c.UpstreamApiEndpoint(), c.Audience, issuerURI)
			if err != nil {
				log.Error().Err(err).Msg("could not exchange external (wif) token")
				return nil
			}

			return serviceAccount
		}
	}

	// return nil when no service account is defined
	if c.ServiceAccountMrn == "" && c.PrivateKey == "" && c.Certificate == "" {
		return nil
	}

	return &upstream.ServiceAccountCredentials{
		Mrn:         c.ServiceAccountMrn,
		ParentMrn:   c.GetScopeMrn(),
		ScopeMrn:    c.GetScopeMrn(),
		PrivateKey:  c.PrivateKey,
		Certificate: c.Certificate,
		ApiEndpoint: c.APIEndpoint,
	}
}

// GetScopeMrn returns the scope mrn that is used for the service account.
// This is either the organization mrn or the space mrn.
func (c *CommonOpts) GetScopeMrn() string {
	scopeMrn := c.ScopeMrn

	// fallback to old space_mrn config
	if scopeMrn == "" {
		scopeMrn = c.SpaceMrn
	}

	if scopeMrn == "" {
		scopeMrn = c.ParentMrn
	}

	return scopeMrn
}

// GetParentMrn returns the scope mrn that is used for the service account.
// This is either the organization mrn or the space mrn.
// Deprecated: Use GetScopeMrn instead
func (c *CommonOpts) GetParentMrn() string {
	return c.GetScopeMrn()
}

func (c *CommonOpts) UpstreamApiEndpoint() string {
	// First check for environment variable override
	if envEndpoint := os.Getenv("MONDOO_API_ENDPOINT"); envEndpoint != "" {
		return envEndpoint
	}

	apiEndpoint := c.APIEndpoint

	// fallback to default api if nothing was set
	if apiEndpoint == "" {
		apiEndpoint = defaultAPIendpoint
	}

	return apiEndpoint
}

// IsWifConfigFormat determines if a file is in the WIF config format
func IsWifConfigFormat(filePath string) (bool, error) {
	// Read the file
	data, err := afero.ReadFile(AppFs, filePath)
	if err != nil {
		return false, err
	}

	// Try to parse it as JSON
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return false, nil // Not JSON, so not WIF format
	}

	// Check if it has the required fields for WIF format
	accountType, hasType := config["type"]
	if !hasType {
		return false, nil
	}

	// Check if it's an external account config
	typeStr, ok := accountType.(string)
	if !ok || typeStr != "external_account" {
		return false, nil
	}

	// Check for audience
	_, hasAudience := config["audience"]

	return hasAudience, nil
}

// ConvertWifConfig reads a WIF config file and converts it to a Viper-compatible format
func ConvertWifConfig(filePath string, v *viper.Viper) error {
	// Read the file
	data, err := afero.ReadFile(AppFs, filePath)
	if err != nil {
		return err
	}

	// Parse it as JSON
	var wifConfig map[string]interface{}
	if err := json.Unmarshal(data, &wifConfig); err != nil {
		return errors.Wrap(err, "failed to parse WIF config as JSON")
	}

	// Set the authentication method
	v.Set("auth", map[string]string{"method": "wif"})

	// Set the required fields
	if audience, ok := wifConfig["audience"].(string); ok {
		v.Set("audience", audience)
	} else {
		return errors.New("WIF config missing required 'audience' field")
	}

	// Check for universeDomain (required for API endpoint)
	if universeDomain, ok := wifConfig["universeDomain"].(string); ok {
		v.Set("universeDomain", universeDomain)
		v.Set("api_endpoint", universeDomain)
		log.Debug().Str("universeDomain", universeDomain).Msg("setting API endpoint from universeDomain")
	} else {
		return errors.New("WIF config missing required 'universeDomain' field")
	}

	if issuerUri, ok := wifConfig["issuerUri"].(string); ok {
		v.Set("issuer_uri", issuerUri)
	} else {
		return errors.New("WIF config missing required 'issuerUri' field")
	}

	// Copy all other fields for consistency
	for key, value := range wifConfig {
		if key != "auth" && key != "audience" && key != "issuer_uri" && key != "api_endpoint" && key != "universeDomain" {
			v.Set(key, value)
		}
	}

	return nil
}
