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
	"context"
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
	"go.mondoo.io/mondoo/motor/discovery/tfstate"
	"go.mondoo.io/mondoo/motor/discovery/vagrant"
	"go.mondoo.io/mondoo/motor/discovery/vsphere"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/providers"
	pr "go.mondoo.io/mondoo/motor/providers/resolver"
	"go.mondoo.io/mondoo/stringx"
)

type Resolver interface {
	Name() string
	Resolve(ctx context.Context, root *asset.Asset, t *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn,
		userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error)
	AvailableDiscoveryTargets() []string
}

var resolver map[string]Resolver

func init() {
	resolver = map[string]Resolver{
		providers.ProviderID_LOCAL:              &local.Resolver{},
		providers.ProviderID_WINRM:              &standard.Resolver{},
		providers.ProviderID_SSH:                &standard.Resolver{},
		providers.ProviderID_DOCKER:             &docker_engine.Resolver{},
		providers.ProviderID_DOCKER_IMAGE:       &docker_engine.Resolver{},
		providers.ProviderID_DOCKER_CONTAINER:   &docker_engine.Resolver{},
		providers.ProviderID_TAR:                &tar.Resolver{},
		providers.ProviderID_K8S:                &k8s.Resolver{},
		providers.ProviderID_GCR:                &gcp.GcrResolver{},
		providers.ProviderID_GCP:                &gcp.GcpResolver{},
		providers.ProviderID_CONTAINER_REGISTRY: &container_registry.Resolver{},
		providers.ProviderID_AZURE:              &azure.Resolver{},
		providers.ProviderID_AWS:                &aws.Resolver{},
		providers.ProviderID_VAGRANT:            &vagrant.Resolver{},
		providers.ProviderID_MOCK:               &mock.Resolver{},
		providers.ProviderID_VSPHERE:            &vsphere.Resolver{},
		providers.ProviderID_VSPHERE_VM:         &vsphere.VMGuestResolver{},
		providers.ProviderID_ARISTA:             &standard.Resolver{},
		providers.ProviderID_MS365:              &ms365.Resolver{},
		providers.ProviderID_IPMI:               &ipmi.Resolver{},
		providers.ProviderID_FS:                 &standard.Resolver{},
		providers.ProviderID_EQUINIX:            &equinix.Resolver{},
		providers.ProviderID_GITHUB:             &github.Resolver{},
		providers.ProviderID_AWS_EC2_EBS:        &ebs.Resolver{},
		providers.ProviderID_GITLAB:             &gitlab.Resolver{},
		providers.ProviderID_TERRAFORM:          &terraform.Resolver{},
		providers.ProviderID_HOST:               &network.Resolver{},
		providers.ProviderID_TLS:                &network.Resolver{},
		providers.ProviderID_TERRAFORM_STATE:    &tfstate.Resolver{},
	}
}

// InitCtx initializes the context to support all resolvers
func InitCtx(ctx context.Context) context.Context {
	initCtx := ctx
	for _, r := range resolver {
		if ctxInitializer, ok := r.(common.ContextInitializer); ok {
			initCtx = ctxInitializer.InitCtx(initCtx)
		}
	}
	return initCtx
}

func ResolveAsset(ctx context.Context, root *asset.Asset, cfn common.CredentialFn, sfn common.QuerySecretFn) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// if the asset is missing a secret, we try to add this for the asset
	common.EnrichAssetWithSecrets(root, sfn)

	assetFallbackName := func(a *asset.Asset, c *providers.Config) {
		// set the asset name to the config name. This is only required for error cases where the discovery
		// is not successful
		if root.Name == "" {
			root.Name = c.Host
		}
	}

	for i := range root.Connections {
		pCfg := root.Connections[i]

		resolverId := pCfg.Backend.Id()
		r, ok := resolver[resolverId]
		if !ok {
			assetFallbackName(root, pCfg)
			return nil, errors.New("unsupported backend: " + resolverId)
		}

		log.Debug().Str("resolver-id", resolverId).Str("resolver", r.Name()).Msg("run resolver")
		// check that all discovery options are supported and show a user warning
		availableTargets := r.AvailableDiscoveryTargets()
		if pCfg.Discover != nil {
			for i := range pCfg.Discover.Targets {
				target := pCfg.Discover.Targets[i]
				if !stringx.Contains(availableTargets, target) {
					log.Warn().Str("resolver", r.Name()).Msgf("resolver does not support discovery target '%s', the following are supported: %s", target, strings.Join(availableTargets, ","))
				}
			}
		}

		userIdDetectors := providers.ToPlatformIdDetectors(root.IdDetector)

		// resolve assets
		resolvedAssets, err := r.Resolve(ctx, root, pCfg, cfn, sfn, userIdDetectors...)
		if err != nil {
			assetFallbackName(root, pCfg)
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
	Assets        []*asset.Asset
	RelatedAssets []*asset.Asset
	Errors        map[*asset.Asset]error
}

func ResolveAssets(ctx context.Context, rootAssets []*asset.Asset, cfn common.CredentialFn, sfn common.QuerySecretFn) ResolvedAssets {
	resolved := []*asset.Asset{}
	resolvedMap := map[string]struct{}{}
	errors := map[*asset.Asset]error{}
	relatedAssets := []*asset.Asset{}
	platformIdToAssetMap := map[string]*asset.Asset{}

	for i := range rootAssets {
		asset := rootAssets[i]

		resolverAssets, err := ResolveAsset(ctx, asset, cfn, sfn)
		if err != nil {
			errors[asset] = err
			continue
		}

		for _, resolvedAsset := range resolverAssets {
			for _, platformId := range resolvedAsset.PlatformIds {
				if platformId != "" {
					platformIdToAssetMap[platformId] = asset
					resolvedMap[platformId] = struct{}{}
				}
			}

			for _, a := range resolvedAsset.RelatedAssets {
				relatedAssets = append(relatedAssets, a)
			}
		}

		resolved = append(resolved, resolverAssets...)
	}

	resolveRelatedAssets(ctx, relatedAssets, platformIdToAssetMap, cfn)

	neededRelatedAssets := []*asset.Asset{}
	for _, a := range relatedAssets {
		found := false
		for _, platformId := range a.PlatformIds {
			if _, ok := resolvedMap[platformId]; ok {
				found = true
				break
			}
		}
		if found {
			continue
		}
		neededRelatedAssets = append(neededRelatedAssets, a)
	}

	return ResolvedAssets{
		Assets:        resolved,
		RelatedAssets: neededRelatedAssets,
		Errors:        errors,
	}
}

func resolveRelatedAssets(ctx context.Context, relatedAssets []*asset.Asset, platformIdToAssetMap map[string]*asset.Asset, cfn common.CredentialFn) {
	for _, assetObj := range relatedAssets {
		if len(assetObj.PlatformIds) > 0 {
			for _, platformId := range assetObj.PlatformIds {
				platformIdToAssetMap[platformId] = assetObj
			}
			continue
		}
		if len(assetObj.Connections) > 0 {
			tc := assetObj.Connections[0]
			if tc.PlatformId != "" {
				assetObj.PlatformIds = []string{tc.PlatformId}
				platformIdToAssetMap[tc.PlatformId] = assetObj
				continue
			}

			func() {
				m, err := pr.NewMotorConnection(ctx, tc, cfn)
				if err != nil {
					log.Warn().Err(err).Msg("could not connect to related asset")
					return
				}
				defer m.Close()
				p, err := m.Platform()
				if err != nil {
					log.Warn().Err(err).Msg("could not get related asset platform")
					return
				}
				fingerprint, err := motorid.IdentifyPlatform(m.Provider, p, m.Provider.PlatformIdDetectors())
				if err != nil {
					return
				}

				assetObj.State = asset.State_STATE_ONLINE
				assetObj.Name = fingerprint.Name
				assetObj.PlatformIds = fingerprint.PlatformIDs
				assetObj.Platform = p

				for _, v := range fingerprint.PlatformIDs {
					platformIdToAssetMap[v] = assetObj
				}
			}()
		}
	}
}
