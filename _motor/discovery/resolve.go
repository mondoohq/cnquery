// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

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
	"go.mondoo.com/cnquery/motor/discovery/common"
	inventory "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers"
	pr "go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/stringx"
)

type Resolver interface {
	Name() string
	Resolve(ctx context.Context, root *inventory.Asset, t *inventory.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn,
		userIdDetectors ...providers.PlatformIdDetector) ([]*inventory.Asset, error)
	AvailableDiscoveryTargets() []string
}

var resolver map[string]Resolver

func init() {
	resolver = map[string]Resolver{}
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

func ResolveAsset(ctx context.Context, root *inventory.Asset, credsResolver vault.Resolver, sfn common.QuerySecretFn) ([]*inventory.Asset, error) {
	resolved := []*inventory.Asset{}

	// if the asset is missing a secret, we try to add this for the asset
	common.EnrichAssetWithSecrets(root, sfn)

	assetFallbackName := func(a *inventory.Asset, c *inventory.Config) {
		// set the asset name to the config name. This is only required for error cases where the discovery
		// is not successful
		if a.Name == "" {
			a.Name = c.Host
		}
	}

	for i := range root.Connections {
		pCfg := root.Connections[i]

		resolverId := pCfg.Type
		r, ok := resolver[resolverId]
		if !ok {
			assetFallbackName(root, pCfg)
			return nil, errors.New("cannot discover backend: " + resolverId)
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
		resolvedAssets, err := r.Resolve(ctx, root, pCfg, credsResolver, sfn, userIdDetectors...)
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

			// copy over managedBy from root
			assetObj.ManagedBy = root.GetManagedBy()

			// if the user set the asset name via flag, --asset-name,
			// that value should override the discovered one
			if root.Name != "" {
				assetObj.Name = root.Name
			}
			resolved = append(resolved, assetObj)
		}
	}
	return resolved, nil
}

type ResolvedAssets struct {
	Assets        []*inventory.Asset
	RelatedAssets []*inventory.Asset
	Errors        map[*inventory.Asset]error
}

func ResolveAssets(ctx context.Context, rootAssets []*inventory.Asset, credsResolver vault.Resolver, sfn common.QuerySecretFn) ResolvedAssets {
	resolved := []*inventory.Asset{}
	resolvedMap := map[string]struct{}{}
	errors := map[*inventory.Asset]error{}
	relatedAssets := []*inventory.Asset{}
	platformIdToAssetMap := map[string]*inventory.Asset{}

	for i := range rootAssets {
		asset := rootAssets[i]

		resolverAssets, err := ResolveAsset(ctx, asset, credsResolver, sfn)
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

	resolveRelatedAssets(ctx, relatedAssets, platformIdToAssetMap, credsResolver)

	neededRelatedAssets := []*inventory.Asset{}
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

func resolveRelatedAssets(ctx context.Context, relatedAssets []*inventory.Asset, platformIdToAssetMap map[string]*inventory.Asset, credsResolver vault.Resolver) {
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
				m, err := pr.NewMotorConnection(ctx, tc, credsResolver)
				if err != nil {
					log.Warn().Err(err).Msg("could not connect to related asset")
					return
				}
				defer m.Close()
				// p, err := m.Platform()
				if err != nil {
					log.Warn().Err(err).Msg("could not get related asset platform")
					return
				}

				panic("REDO")
				// fingerprint, err := motorid.IdentifyPlatform(m.Provider, p, m.Provider.PlatformIdDetectors())
				// if err != nil {
				// 	return
				// }

				// if fingerprint.Runtime != "" {
				// 	p.Runtime = fingerprint.Runtime
				// }

				// if fingerprint.Kind != providers.Kind_KIND_UNKNOWN {
				// 	p.Kind = fingerprint.Kind
				// }

				// assetObj.State = asset.State_STATE_ONLINE
				// assetObj.Name = fingerprint.Name
				// assetObj.PlatformIds = fingerprint.PlatformIDs
				// assetObj.Platform = p

				// for _, v := range fingerprint.PlatformIDs {
				// 	platformIdToAssetMap[v] = assetObj
				// }
			}()
		}
	}
}
