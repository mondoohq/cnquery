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
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/discovery/container_registry"
	"go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/discovery/equinix"
	"go.mondoo.io/mondoo/motor/discovery/gcp"
	"go.mondoo.io/mondoo/motor/discovery/ipmi"
	"go.mondoo.io/mondoo/motor/discovery/k8s"
	"go.mondoo.io/mondoo/motor/discovery/local"
	"go.mondoo.io/mondoo/motor/discovery/mock"
	"go.mondoo.io/mondoo/motor/discovery/ms365"
	"go.mondoo.io/mondoo/motor/discovery/standard"
	"go.mondoo.io/mondoo/motor/discovery/tar"
	"go.mondoo.io/mondoo/motor/discovery/vagrant"
	"go.mondoo.io/mondoo/motor/discovery/vsphere"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/stringx"
)

type Resolver interface {
	Name() string
	Resolve(t *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn,
		userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error)
	AvailableDiscoveryTargets() []string
}

var resolver map[string]Resolver

func init() {
	resolver = map[string]Resolver{
		transports.SCHEME_LOCAL:              &local.Resolver{},
		transports.SCHEME_WINRM:              &standard.Resolver{},
		transports.SCHEME_SSH:                &standard.Resolver{},
		transports.SCHEME_DOCKER:             &docker_engine.Resolver{},
		transports.SCHEME_DOCKER_IMAGE:       &docker_engine.Resolver{},
		transports.SCHEME_DOCKER_CONTAINER:   &docker_engine.Resolver{},
		transports.SCHEME_TAR:                &tar.Resolver{},
		transports.SCHEME_K8S:                &k8s.Resolver{},
		transports.SCHEME_GCR:                &gcp.GcrResolver{},
		transports.SCHEME_GCP:                &gcp.GcpResolver{},
		transports.SCHEME_CONTAINER_REGISTRY: &container_registry.Resolver{},
		transports.SCHEME_AZURE:              &azure.Resolver{},
		transports.SCHEME_AWS:                &aws.Resolver{},
		transports.SCHEME_VAGRANT:            &vagrant.Resolver{},
		transports.SCHEME_MOCK:               &mock.Resolver{},
		transports.SCHEME_VSPHERE:            &vsphere.Resolver{},
		transports.SCHEME_VSPHERE_VM:         &vsphere.VMGuestResolver{},
		transports.SCHEME_ARISTA:             &standard.Resolver{},
		transports.SCHEME_MS365:              &ms365.Resolver{},
		transports.SCHEME_IPMI:               &ipmi.Resolver{},
		transports.SCHEME_FS:                 &standard.Resolver{},
		transports.SCHEME_EQUINIX:            &equinix.Resolver{},
		transports.SCHEME_GITHUB:             &standard.Resolver{},
		transports.SCHEME_AWS_EC2_EBS:        &ebs.Resolver{},
		transports.SCHEME_GITLAB:             &standard.Resolver{},
		transports.SCHEME_TERRAFORM:          &standard.Resolver{},
	}
}

func ResolveAsset(root *asset.Asset, cfn common.CredentialFn, sfn common.QuerySecretFn) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// if the asset is missing a secret, we try to add this for the asset
	common.EnrichAssetWithSecrets(root, sfn)

	for i := range root.Connections {
		tc := root.Connections[i]

		resolverId := tc.Backend.Scheme()
		r, ok := resolver[resolverId]
		if !ok {
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

		userIdDetectors := transports.ToPlatformIdDetectors(root.IdDetector)

		// resolve assets
		resolvedAssets, err := r.Resolve(tc, cfn, sfn, userIdDetectors...)
		if err != nil {
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

			resolved = append(resolved, assetObj)
		}
	}
	return resolved, nil
}

type ResolvedAssets struct {
	Assets []*asset.Asset
	Errors map[*asset.Asset]error
}

func ResolveAssets(rootAssets []*asset.Asset, cfn common.CredentialFn, sfn common.QuerySecretFn) ResolvedAssets {
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
