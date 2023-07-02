package config

import (
	"errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/upstream"
)

const defaultAPIendpoint = "https://us.api.mondoo.com"

func ReadConfig() (*CliConfig, error) {
	// load viper config into a struct
	var opts CliConfig
	err := viper.Unmarshal(&opts)
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to decode into config struct"))
	}

	return &opts, nil
}

type CliConfig struct {
	// inherit common config
	CommonCliConfig `mapstructure:",squash"`

	// Asset Category
	Category               string `json:"category,omitempty" mapstructure:"category"`
	AutoDetectCICDCategory bool   `json:"detect-cicd,omitempty" mapstructure:"detect-cicd"`
}

type CommonCliConfig struct {
	// client identifier
	AgentMrn string `json:"agent_mrn,omitempty" mapstructure:"agent_mrn"`

	// service account credentials
	ServiceAccountMrn string `json:"mrn,omitempty" mapstructure:"mrn"`
	ParentMrn         string `json:"parent_mrn,omitempty" mapstructure:"parent_mrn"`
	SpaceMrn          string `json:"space_mrn,omitempty" mapstructure:"space_mrn"`
	PrivateKey        string `json:"private_key,omitempty" mapstructure:"private_key"`
	Certificate       string `json:"certificate,omitempty" mapstructure:"certificate"`
	APIEndpoint       string `json:"api_endpoint,omitempty" mapstructure:"api_endpoint"`

	// authentication
	Authentication *CliConfigAuthentication `json:"auth,omitempty" mapstructure:"auth"`

	// client features
	Features []string `json:"features,omitempty" mapstructure:"features"`

	// API Proxy for communicating with Mondoo API
	APIProxy string `json:"api_proxy,omitempty" mapstructure:"api_proxy"`

	// labels that will be applied to all assets
	Labels map[string]string `json:"labels,omitempty" mapstructure:"labels"`
}

type CliConfigAuthentication struct {
	Method string `json:"method,omitempty" mapstructure:"method"`
}

func (c *CommonCliConfig) GetFeatures() cnquery.Features {
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
			log.Warn().Str("feature", name).Msg("could not parse feature")
		}
	}

	return flags
}

// GetServiceCredential returns the service credential that is defined in the config.
// If no service credential is defined, it will return nil.
func (c *CommonCliConfig) GetServiceCredential() *upstream.ServiceAccountCredentials {
	if c.Authentication != nil && c.Authentication.Method == "ssh" {
		log.Info().Msg("using ssh authentication method, generate temporary credentials")
		serviceAccount, err := upstream.ExchangeSSHKey(c.UpstreamApiEndpoint(), c.ServiceAccountMrn, c.GetParentMrn())
		if err != nil {
			log.Error().Err(err).Msg("could not exchange ssh key")
			return nil
		}
		return serviceAccount
	}

	// return nil when no service account is defined
	if c.ServiceAccountMrn == "" && c.PrivateKey == "" && c.Certificate == "" {
		return nil
	}

	return &upstream.ServiceAccountCredentials{
		Mrn:         c.ServiceAccountMrn,
		ParentMrn:   c.GetParentMrn(),
		PrivateKey:  c.PrivateKey,
		Certificate: c.Certificate,
		ApiEndpoint: c.APIEndpoint,
	}
}

func (c *CommonCliConfig) GetParentMrn() string {
	parent := c.ParentMrn

	// fallback to old space_mrn config
	if parent == "" {
		parent = c.SpaceMrn
	}

	return parent
}

func (c *CommonCliConfig) UpstreamApiEndpoint() string {
	apiEndpoint := c.APIEndpoint

	// fallback to default api if nothing was set
	if apiEndpoint == "" {
		apiEndpoint = defaultAPIendpoint
	}

	return apiEndpoint
}
