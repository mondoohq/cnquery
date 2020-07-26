package resolver

import (
	"strings"

	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/aws"
)

type Ec2Config struct {
	User    string
	Region  string
	Profile string
}

func ParseEc2InstanceContext(ec2Url string) Ec2Config {
	var config Ec2Config

	ec2Url = strings.TrimPrefix(ec2Url, "ec2://")

	keyValues := strings.Split(ec2Url, "/")
	for i := 0; i < len(keyValues); {
		if keyValues[i] == "user" {
			if i+1 < len(keyValues) {
				config.User = keyValues[i+1]
			}
		}
		if keyValues[i] == "region" {
			if i+1 < len(keyValues) {
				config.Region = keyValues[i+1]
			}
		}
		if keyValues[i] == "profile" {
			if i+1 < len(keyValues) {
				config.Profile = keyValues[i+1]
			}
		}
		i = i + 2
	}

	return config
}

type awsResolver struct{}

func (k *awsResolver) Resolve(in *options.VulnOptsAsset, opts *options.VulnOpts) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// parse context from url
	config := ParseEc2InstanceContext(in.Connection)

	configs := []external.Config{}
	if len(config.Profile) > 0 {
		configs = append(configs, external.WithSharedConfigProfile(config.Profile))
	}

	cfg, err := external.LoadDefaultAWSConfig(configs...)
	if err != nil {
		return nil, errors.Wrap(err, "could not load aws configuration")
	}

	if len(config.Region) > 0 {
		cfg.Region = config.Region
	}

	r, err := aws.NewEc2Discovery(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize aws ec2 discovery")
	}

	// we may want to pass a specific user, otherwise it will fallback to ssh config
	if len(config.User) > 0 {
		r.InstanceSSHUsername = config.User
	}

	assetList, err := r.List()
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch ec2 instances")
	}
	log.Debug().Int("instances", len(assetList)).Msg("completed instance search")
	for i := range assetList {
		log.Debug().Str("name", assetList[i].Name).Msg("resolved ec2 instance")
		if assetList[i].State != asset.State_STATE_RUNNING {
			log.Warn().Str("name", assetList[i].Name).Msg("skip instance that is not running")
			continue
		}
		resolved = append(resolved, assetList[i])
	}

	return resolved, nil
}
