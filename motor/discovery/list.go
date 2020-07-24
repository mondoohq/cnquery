package discovery

import (
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/aws"
	"go.mondoo.io/mondoo/motor/discovery/docker"
	"go.mondoo.io/mondoo/motor/discovery/gcp"
	"go.mondoo.io/mondoo/motor/stringslice"
	"go.mondoo.io/mondoo/nexus/assets"
)

type Plugin interface {
	List() ([]*assets.Asset, error)
}

func ListAssets(runtimes ...string) ([]*assets.Asset, error) {
	askRuntimes := []Plugin{}

	if stringslice.Contains(runtimes, asset.RUNTIME_AWS_EC2) ||
		stringslice.Contains(runtimes, asset.RUNTIME_AWS_SSM_MANAGED) ||
		stringslice.Contains(runtimes, asset.RUNTIME_AWS_ECR) {
		cfg, err := external.LoadDefaultAWSConfig()
		if err != nil {
			log.Warn().Err(err).Msg("skip aws assets")
		} else {
			if stringslice.Contains(runtimes, asset.RUNTIME_AWS_EC2) {
				plugin_aws, err := aws.NewEc2Discovery(cfg)
				if err == nil {
					askRuntimes = append(askRuntimes, plugin_aws)
				}
			}

			if stringslice.Contains(runtimes, asset.RUNTIME_AWS_SSM_MANAGED) {
				plugin_aws, err := aws.NewSSMManagedInstancesDiscovery(cfg)
				if err == nil {
					askRuntimes = append(askRuntimes, plugin_aws)
				}
			}

			if stringslice.Contains(runtimes, asset.RUNTIME_AWS_ECR) {
				plugin_aws, err := aws.NewEcrImages(cfg)
				if err == nil {
					askRuntimes = append(askRuntimes, plugin_aws)
				}
			}
		}
	}

	// if stringslice.Contains(runtimes, asset.RUNTIME_GCP_COMPUTE) {
	// 	askRuntimes = append(askRuntimes, gcp.NewCompute())
	// }

	if stringslice.Contains(runtimes, asset.RUNTIME_GCP_GCR) {
		askRuntimes = append(askRuntimes, gcp.NewGCRImages())
	}

	if stringslice.Contains(runtimes, asset.RUNTIME_DOCKER_CONTAINER) {
		askRuntimes = append(askRuntimes, &docker.Container{})
	}

	if stringslice.Contains(runtimes, asset.RUNTIME_DOCKER_IMAGE) {
		askRuntimes = append(askRuntimes, &docker.Images{})
	}

	// if stringslice.Contains(runtimes, asset.RUNTIME_DOCKER_REGISTRY) {
	// 	askRuntimes = append(askRuntimes, &docker.DockerRegistryImages{})
	// }

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
