package aws

import (
	"github.com/pkg/errors"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

func New(tc *transports.TransportConfig) (*Transport, error) {
	if tc.Backend != transports.TransportBackend_CONNECTION_AWS {
		return nil, errors.New("backend is not supported for aws transport")
	}

	configs := []external.Config{}
	if tc.Options != nil && len(tc.Options["profile"]) > 0 {
		configs = append(configs, external.WithSharedConfigProfile(tc.Options["profile"]))
	}

	cfg, err := external.LoadDefaultAWSConfig(configs...)
	if err != nil {
		return nil, errors.Wrap(err, "could not load aws configuration")
	}

	if tc.Options != nil && len(tc.Options["region"]) > 0 {
		cfg.Region = tc.Options["region"]
	}

	identity, err := CheckIam(cfg)
	if err != nil {
		return nil, err
	}

	return &Transport{
		config: cfg,
		opts:   tc.Options,
		info: Info{
			Account: toString(identity.Account),
			Arn:     toString(identity.Arn),
			UserId:  toString(identity.UserId),
		},
	}, nil
}

func toString(i *string) string {
	if i == nil {
		return ""
	}
	return *i
}

type Info struct {
	Account string
	Arn     string
	UserId  string
}

type Transport struct {
	config             aws_sdk.Config
	opts               map[string]string
	selectedPlatformID string
	info               Info
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("vsphere does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("vsphere does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{}
}

func (t *Transport) Config() aws_sdk.Config {
	return t.config
}

func (t *Transport) Options() map[string]string {
	return t.opts
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return transports.RUNTIME_AWS
}
