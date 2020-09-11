package discovery

import (
	"strings"

	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/aws"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

type Ec2Config struct {
	User    string
	Region  string
	Profile string
}

func ParseAwsContext(awsUrl string) Ec2Config {
	var config Ec2Config

	awsUrl = strings.TrimPrefix(awsUrl, "aws://")
	awsUrl = strings.TrimPrefix(awsUrl, "ec2://")

	keyValues := strings.Split(awsUrl, "/")
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

func (k *awsResolver) Name() string {
	return "AWS EC2 Resolver"
}

func (k *awsResolver) Resolve(in *options.VulnOptsAsset, opts *options.VulnOpts) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// parse context from url
	config := ParseAwsContext(in.Connection)

	t := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_AWS,
		Options: map[string]string{
			"profile": config.Profile,
			"region":  config.Region,
		},
	}

	// add aws api as asset
	trans, err := aws_transport.New(t)
	if err != nil {
		return nil, err
	}

	identifier, err := trans.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := platform.NewDetector(trans)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	// add asset for the api itself
	info, err := trans.Account()
	if err != nil {
		return nil, err
	}

	resolved = append(resolved, &asset.Asset{
		ReferenceIDs: []string{identifier},
		Name:         "AWS Account " + info.Name + "(" + info.ID + ")",
		Platform:     pf,
		Connections:  []*transports.TransportConfig{t}, // pass-in the current config
	})

	// discover ec2 instances
	if opts.DiscoverInstances {
		// TODO: rewrite ec2 discovert to use the aws transport
		r, err := aws.NewEc2Discovery(trans.Config())
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
	}

	return resolved, nil
}
