package config

import (
	"context"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"

	"errors"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const AWS_SSM_PARAMETERSTORE_PREFIX = "aws-ssm-ps://"

// loads the configuration from aws ssm parameter store
func loadAwsSSMParameterStore(key string) error {
	viper.RemoteConfig = &awsSSMParamConfigFactory{}
	viper.SupportedRemoteProviders = []string{"aws-ssm-ps"}
	ssmKey := strings.TrimPrefix(key, AWS_SSM_PARAMETERSTORE_PREFIX)
	log.Info().Str("key", ssmKey).Msg("look for configuration stored in aws ssm parameter store")
	err := viper.AddRemoteProvider("aws-ssm-ps", "localhost", ssmKey)
	if err != nil {
		return errors.Join(err, errors.New("could not initialize gs provider"))
	}
	viper.SetConfigType("yaml")
	err = viper.ReadRemoteConfig()
	if err != nil {
		return errors.Join(err, errors.New(fmt.Sprintf("could not read aws ssm parameter config from %s", ssmKey)))
	}

	return nil
}

type awsSSMParamConfigFactory struct{}

func (a *awsSSMParamConfigFactory) Get(rp viper.RemoteProvider) (io.Reader, error) {
	ssmParameter, err := ParseSSMParameterPath(rp.Path())
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	cfg.Region = ssmParameter.Region
	ps := ssm.NewFromConfig(cfg)

	out, err := ps.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(ssmParameter.Parameter),
		WithDecryption: aws.Bool(true), // this field is ignored if the parameter is a string or stringlist, so it's ok to have it on by default
	})
	if err != nil {
		return nil, err
	}

	return strings.NewReader(*out.Parameter.Value), nil
}

func (g *awsSSMParamConfigFactory) Watch(rp viper.RemoteProvider) (io.Reader, error) {
	return nil, errors.New("not implemented")
}

func (g *awsSSMParamConfigFactory) WatchChannel(rp viper.RemoteProvider) (<-chan *viper.RemoteResponse, chan bool) {
	return nil, nil
}

type SsmParameter struct {
	Parameter string
	Region    string
	// todo: add optional decrypt and account arguments
}

func NewSSMParameter(region string, parameter string) (*SsmParameter, error) {
	if region == "" || parameter == "" {
		return nil, errors.New("invalid parameter. region and parameter name required.")
	}
	return &SsmParameter{Region: region, Parameter: parameter}, nil
}

func (s *SsmParameter) String() string {
	// e.g. region/us-east-2/parameter/MondooAgentConfig
	return path.Join("region", s.Region, "parameter", s.Parameter)
}

func ParseSSMParameterPath(path string) (*SsmParameter, error) {
	if !IsValidSSMParameterPath(path) {
		return nil, errors.New("invalid parameter path. expected region/<region-val>/parameter/<parameter-name>")
	}
	keyValues := strings.Split(path, "/")
	if len(keyValues) != 4 {
		return nil, errors.New("invalid parameter path. expected region/<region-val>/parameter/<parameter-name>")
	}
	return NewSSMParameter(keyValues[1], keyValues[3])
}

var VALID_SSM_PARAMETER_PATH = regexp.MustCompile(`^region\/(us(-gov)?|ap|ca|cn|eu|sa)-(central|(north|south)?(east|west)?)-\d\/parameter\/.+$`)

func IsValidSSMParameterPath(path string) bool {
	return VALID_SSM_PARAMETER_PATH.MatchString(path)
}
