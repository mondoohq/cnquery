package k8s

import (
	"context"
	"strings"

	"github.com/gobwas/glob"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
	"go.mondoo.com/cnquery/motor/vault"
	"k8s.io/apimachinery/pkg/api/errors"
)

var _ common.ContextInitializer = (*NamespaceResolver)(nil)

type NamespaceResolver struct{}

func (r *NamespaceResolver) Name() string {
	return "Kubernetes Namespace Resolver"
}

func (r *NamespaceResolver) AvailableDiscoveryTargets() []string {
	return []string{
		common.DiscoveryAuto,
		common.DiscoveryAll,
		DiscoveryClusters,
		DiscoveryPods,
		DiscoveryJobs,
		DiscoveryCronJobs,
		DiscoveryStatefulSets,
		DiscoveryDeployments,
		DiscoveryReplicaSets,
		DiscoveryDaemonSets,
		DiscoveryContainerImages,
		DiscoveryAdmissionReviews,
		DiscoveryIngresses,
	}
}

func (r *NamespaceResolver) InitCtx(ctx context.Context) context.Context {
	return resources.SetDiscoveryCache(ctx, resources.NewDiscoveryCache())
}

func (r *NamespaceResolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	features := cnquery.GetFeatures(ctx)
	resolved := []*asset.Asset{}

	nsFilter := NamespaceFilterOpts{}
	includeNamespaces := tc.Options["namespaces"]
	if len(includeNamespaces) > 0 {
		nsFilter.include = append(nsFilter.include, strings.Split(includeNamespaces, ",")...)
	}

	p, err := k8s.New(ctx, tc)
	if err != nil {
		return nil, err
	}

	// Put a Warn() message if a Namespace that doesn't exist was part of the
	// list of Namespaces to include. We can only check that if the k8s user is allowed to list the
	// cluster namespaces.
	clusterNamespaces, err := p.Namespaces()
	if err != nil {
		if errors.IsForbidden(err) {
			log.Warn().Msg("cannot list cluster namespaces, skipping check for non-existent namespaces...")
		} else {
			return nil, err
		}
	} else {
		for _, ns := range nsFilter.include {
			foundNamespace := false
			g, err := glob.Compile(ns)
			if err != nil {
				log.Error().Err(err).Str("namespaceFilter", ns).Msg("failed to parse Namespace filter glob")
				return nil, err
			}
			for _, clusterNs := range clusterNamespaces {
				if g.Match(clusterNs.Name) {
					foundNamespace = true
					break
				}
			}
			if !foundNamespace {
				log.Warn().Msgf("Namespace filter %q did not match any Namespaces in cluster", ns)
			}
		}
	}

	excludeNamespaces := tc.Options["namespaces-exclude"]
	if len(excludeNamespaces) > 0 {
		nsFilter.exclude = strings.Split(excludeNamespaces, ",")
	}

	log.Debug().Strs("namespacesIncludeFilter", nsFilter.include).Strs("namespacesExcludeFilter", nsFilter.exclude).Msg("resolve k8s assets")

	resourcesFilter, err := resourceFilters(tc)
	if err != nil {
		return nil, err
	}

	nss, err := ListNamespaces(p, tc, "", nsFilter, resourcesFilter, nil)
	if err != nil {
		return nil, err
	}

	resolved = append(resolved, nss...)
	for _, ns := range nss {
		identifier := ns.PlatformIds[0]
		ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(identifier)
		additionalAssets, err := addSeparateAssets(tc, p, nsFilter, resourcesFilter, identifier, ownershipDir, features)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, additionalAssets...)
	}

	return resolved, nil
}
