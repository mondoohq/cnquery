package discover

import (
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/motorcloud/aws"
	"go.mondoo.io/mondoo/motor/motorcloud/docker"
	"go.mondoo.io/mondoo/motor/motorcloud/gcp"
	"go.mondoo.io/mondoo/motor/stringslice"
	"go.mondoo.io/mondoo/nexus/assets"
)

type Plugin interface {
	List() ([]*assets.Asset, error)
}

const (
	RUNTIME_AWS_EC2         = "aws ec2"
	RUNTIME_AWS_SSM_MANAGED = "aws ssm-managed"
	RUNTIME_GCP_COMPUTE     = "gcp compute"
	RUNTIME_DOCKER          = "docker"
)

func ListAssets(runtimes ...string) ([]*assets.Asset, error) {
	askRuntimes := []Plugin{}

	if stringslice.Contains(runtimes, RUNTIME_AWS_EC2) {
		cfg, err := external.LoadDefaultAWSConfig()
		if err != nil {
			log.Warn().Err(err).Msg("skip aws assets")
		} else {
			plugin_aws, err := aws.NewEc2Discovery(cfg)
			if err == nil {
				askRuntimes = append(askRuntimes, plugin_aws)
			}
		}
	}

	if stringslice.Contains(runtimes, RUNTIME_AWS_SSM_MANAGED) {
		cfg, err := external.LoadDefaultAWSConfig()
		if err != nil {
			log.Warn().Err(err).Msg("skip aws assets")
		} else {
			plugin_aws, err := aws.NewSSMManagedInstancesDiscovery(cfg)
			if err == nil {
				askRuntimes = append(askRuntimes, plugin_aws)
			}
		}
	}

	if stringslice.Contains(runtimes, RUNTIME_GCP_COMPUTE) {
		askRuntimes = append(askRuntimes, gcp.New())
	}

	if stringslice.Contains(runtimes, RUNTIME_DOCKER) {
		askRuntimes = append(askRuntimes, &docker.Images{})
		askRuntimes = append(askRuntimes, &docker.Container{})
	}

	discoveredAssets := []*assets.Asset{}
	for i := range askRuntimes {
		pluginAssets, err := askRuntimes[i].List()
		if err == nil {
			discoveredAssets = append(discoveredAssets, pluginAssets...)
		} else {
			// TODO: write plugin name
			log.Error().Err(err).Msg("could not load assets from plugin")
		}
	}

	return discoveredAssets, nil
}
