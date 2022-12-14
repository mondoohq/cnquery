package config

import (
	"github.com/cockroachdb/errors"
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
		return nil, errors.Wrap(err, "unable to decode into config struct")
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

	// client features
	Features []string `json:"features,omitempty" mapstructure:"features"`

	// labels that will be applied to all assets
	Labels map[string]string `json:"labels,omitempty" mapstructure:"labels"`
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
// If no service credential is defined, it will return an nil.
func (v *CommonCliConfig) GetServiceCredential() *upstream.ServiceAccountCredentials {
	// return nil when no service account is defined
	if v.ServiceAccountMrn == "" && v.PrivateKey == "" && v.Certificate == "" {
		return nil
	}

	return &upstream.ServiceAccountCredentials{
		Mrn:         v.ServiceAccountMrn,
		ParentMrn:   v.GetParentMrn(),
		PrivateKey:  v.PrivateKey,
		Certificate: v.Certificate,
		ApiEndpoint: v.APIEndpoint,
	}
}

func (o *CommonCliConfig) GetParentMrn() string {
	parent := o.ParentMrn

	// fallback to old space_mrn config
	if parent == "" {
		parent = o.SpaceMrn
	}

	return parent
}

func (o *CommonCliConfig) UpstreamApiEndpoint() string {
	apiEndpoint := o.APIEndpoint

	// fallback to default api if nothing was set
	if apiEndpoint == "" {
		apiEndpoint = defaultAPIendpoint
	}

	return apiEndpoint
}
