package discovery

// The discovery package is responsible to determine all assets reachable. E.g. If you provide an AWS
// connection, multiple assets like EC2, ECR images as well as EKS clusters can be determined automatically
//
// This package implements all the resolution steps and returns a fully resolved list of assets that mondoo
// can connect to.
//
// As part of the discovery process, secrets need to be determined. This package is designed to have know
// no knowledge about inventory or vault packages. It defines two `common.CredentialFn` and `common.QuerySecretFn`
// to retrieve the required information. The inventory manager injects the correct functions upon initialization

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/aws"
	"go.mondoo.io/mondoo/motor/discovery/aws/ebs"
	"go.mondoo.io/mondoo/motor/discovery/azure"
	"go.mondoo.io/mondoo/motor/discovery/container_registry"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/discovery/equinix"
	"go.mondoo.io/mondoo/motor/discovery/gcp"
	"go.mondoo.io/mondoo/motor/discovery/github"
	"go.mondoo.io/mondoo/motor/discovery/gitlab"
	"go.mondoo.io/mondoo/motor/discovery/ipmi"
	"go.mondoo.io/mondoo/motor/discovery/k8s"
	"go.mondoo.io/mondoo/motor/discovery/local"
	"go.mondoo.io/mondoo/motor/discovery/mock"
	"go.mondoo.io/mondoo/motor/discovery/ms365"
	"go.mondoo.io/mondoo/motor/discovery/network"
	"go.mondoo.io/mondoo/motor/discovery/standard"
	"go.mondoo.io/mondoo/motor/discovery/tar"
	"go.mondoo.io/mondoo/motor/discovery/terraform"
	"go.mondoo.io/mondoo/motor/discovery/vagrant"
	"go.mondoo.io/mondoo/motor/discovery/vsphere"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/stringx"
)

type Resolver interface {
	Name() string
	Resolve(root *asset.Asset, t *providers.TransportConfig, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn,
		userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error)
	AvailableDiscoveryTargets() []string
}

var resolver map[string]Resolver

func init() {
	resolver = map[string]Resolver{
		providers.SCHEME_LOCAL:              &local.Resolver{},
		providers.SCHEME_WINRM:              &standard.Resolver{},
		providers.SCHEME_SSH:                &standard.Resolver{},
		providers.SCHEME_DOCKER:             &docker_engine.Resolver{},
		providers.SCHEME_DOCKER_IMAGE:       &docker_engine.Resolver{},
		providers.SCHEME_DOCKER_CONTAINER:   &docker_engine.Resolver{},
		providers.SCHEME_TAR:                &tar.Resolver{},
		providers.SCHEME_K8S:                &k8s.Resolver{},
		providers.SCHEME_GCR:                &gcp.GcrResolver{},
		providers.SCHEME_GCP:                &gcp.GcpResolver{},
		providers.SCHEME_CONTAINER_REGISTRY: &container_registry.Resolver{},
		providers.SCHEME_AZURE:              &azure.Resolver{},
		providers.SCHEME_AWS:                &aws.Resolver{},
		providers.SCHEME_VAGRANT:            &vagrant.Resolver{},
		providers.SCHEME_MOCK:               &mock.Resolver{},
		providers.SCHEME_VSPHERE:            &vsphere.Resolver{},
		providers.SCHEME_VSPHERE_VM:         &vsphere.VMGuestResolver{},
		providers.SCHEME_ARISTA:             &standard.Resolver{},
		providers.SCHEME_MS365:              &ms365.Resolver{},
		providers.SCHEME_IPMI:               &ipmi.Resolver{},
		providers.SCHEME_FS:                 &standard.Resolver{},
		providers.SCHEME_EQUINIX:            &equinix.Resolver{},
		providers.SCHEME_GITHUB:             &github.Resolver{},
		providers.SCHEME_AWS_EC2_EBS:        &ebs.Resolver{},
		providers.SCHEME_GITLAB:             &gitlab.Resolver{},
		providers.SCHEME_TERRAFORM:          &terraform.Resolver{},
		providers.SCHEME_HOST:               &network.Resolver{},
		providers.SCHEME_TLS:                &network.Resolver{},
	}
}

func ResolveAsset(root *asset.Asset, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// if the asset is missing a secret, we try to add this for the asset
	credentials.EnrichAssetWithSecrets(root, sfn)

	assetFallbackName := func(a *asset.Asset, c *providers.TransportConfig) {
		// set the asset name to the config name. This is only required for error cases where the discovery
		// is not successful
		if root.Name == "" {
			root.Name = c.Host
		}
	}

	for i := range root.Connections {
		tc := root.Connections[i]

		resolverId := tc.Backend.Scheme()
		r, ok := resolver[resolverId]
		if !ok {
			assetFallbackName(root, tc)
			return nil, errors.New("unsupported backend: " + resolverId)
		}

		log.Debug().Str("resolver-id", resolverId).Str("resolver", r.Name()).Msg("run resolver")
		// check that all discovery options are supported and show a user warning
		availableTargets := r.AvailableDiscoveryTargets()
		if tc.Discover != nil {
			for i := range tc.Discover.Targets {
				target := tc.Discover.Targets[i]
				if !stringx.Contains(availableTargets, target) {
					log.Warn().Str("resolver", r.Name()).Msgf("resolver does not support discovery target '%s', the following are supported: %s", target, strings.Join(availableTargets, ","))
				}
			}
		}

		userIdDetectors := providers.ToPlatformIdDetectors(root.IdDetector)

		// resolve assets
		resolvedAssets, err := r.Resolve(root, tc, cfn, sfn, userIdDetectors...)
		if err != nil {
			assetFallbackName(root, tc)
			return nil, err
		}

		for ai := range resolvedAssets {
			assetObj := resolvedAssets[ai]

			// copy over id detector overwrite
			assetObj.IdDetector = root.IdDetector

			// copy over labels from root
			if assetObj.Labels == nil {
				assetObj.Labels = map[string]string{}
			}

			for k, v := range root.Labels {
				assetObj.Labels[k] = v
			}

			// copy over annotations from root
			if assetObj.Annotations == nil {
				assetObj.Annotations = map[string]string{}
			}

			for k, v := range root.Annotations {
				assetObj.Annotations[k] = v
			}
			assetObj.Category = root.Category

			resolved = append(resolved, assetObj)
		}
	}
	return resolved, nil
}

type ResolvedAssets struct {
	Assets []*asset.Asset
	Errors map[*asset.Asset]error
}

func ResolveAssets(rootAssets []*asset.Asset, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn) ResolvedAssets {
	resolved := []*asset.Asset{}
	errors := map[*asset.Asset]error{}
	for i := range rootAssets {
		asset := rootAssets[i]

		resolverAssets, err := ResolveAsset(asset, cfn, sfn)
		if err != nil {
			errors[asset] = err
		}

		resolved = append(resolved, resolverAssets...)
	}

	return ResolvedAssets{
		Assets: resolved,
		Errors: errors,
	}
}
