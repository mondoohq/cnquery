// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package k8s

import (
	"context"

	"go.mondoo.com/cnquery/v9/motor/asset"
	"go.mondoo.com/cnquery/v9/motor/discovery/common"
	"go.mondoo.com/cnquery/v9/motor/providers"
	"go.mondoo.com/cnquery/v9/motor/providers/k8s/resources"
	"go.mondoo.com/cnquery/v9/motor/vault"
)

var _ common.ContextInitializer = (*NamespaceResolver)(nil)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Kubernetes Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
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
		DiscoveryNamespaces,
	}
}

func (r *Resolver) InitCtx(ctx context.Context) context.Context {
	return resources.SetDiscoveryCache(ctx, resources.NewDiscoveryCache())
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	nsFilter := tc.Options["namespaces"]
	if len(nsFilter) > 0 {
		return (&NamespaceResolver{}).Resolve(ctx, root, tc, credsResolver, sfn, userIdDetectors...)
	}
	return (&ClusterResolver{}).Resolve(ctx, root, tc, credsResolver, sfn, userIdDetectors...)
}
