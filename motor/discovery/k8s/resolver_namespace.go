package k8s

import (
	"context"
	"strings"

	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
	"go.mondoo.com/cnquery/motor/vault"
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

	resourcesFilter, err := resourceFilters(tc)
	if err != nil {
		return nil, err
	}

	p, err := k8s.New(ctx, tc)
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
