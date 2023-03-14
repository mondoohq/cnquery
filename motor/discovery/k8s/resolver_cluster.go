package k8s

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/resources/packs/os/kubectl"
)

var _ common.ContextInitializer = (*ClusterResolver)(nil)

type ClusterResolver struct{}

func (r *ClusterResolver) Name() string {
	return "Kubernetes Cluster Resolver"
}

func (r *ClusterResolver) AvailableDiscoveryTargets() []string {
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

func (r *ClusterResolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	features := cnquery.GetFeatures(ctx)
	resolved := []*asset.Asset{}

	var k8sctlConfig *kubectl.KubectlConfig
	localProvider, err := local.New()
	if err == nil {
		k8sctlConfig, err = kubectl.LoadKubeConfig(localProvider)
		if err != nil {
			return nil, err
		}
	}

	p, err := k8s.New(ctx, tc)
	if err != nil {
		return nil, err
	}

	resourcesFilter, err := resourceFilters(tc)
	if err != nil {
		return nil, err
	}

	if tc.IncludesDiscoveryTarget(common.DiscoveryAuto) {
		log.Info().Msg("discovery option auto is used. This will detect the assets: cluster, jobs, cronjobs, pods, statefulsets, deployments, replicasets, daemonsets")
	}

	clusterIdentifier, err := p.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := detector.New(p)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	// Only discover cluster and nodes if there are no resource filters.
	var clusterAsset *asset.Asset
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryClusters) &&
		len(resourcesFilter) == 0 {
		// the name is still a bit unreliable
		// see https://github.com/kubernetes/kubernetes/issues/44954
		clusterName := ""

		if tc.Options[k8s.OPTION_MANIFEST] != "" || tc.Options[k8s.OPTION_ADMISSION] != "" {
			clusterName, _ = p.Name()
		} else {
			// try to parse context from kubectl config
			if clusterName == "" && k8sctlConfig != nil && len(k8sctlConfig.CurrentContext) > 0 {
				clusterName = k8sctlConfig.CurrentClusterName()
				log.Info().Str("cluster-name", clusterName).Msg("use cluster name from kube config")
			}

			// fallback to first node name if we could not gather the name from kubeconfig
			if clusterName == "" {
				name, err := p.Name()
				if err == nil {
					clusterName = name
					log.Info().Str("cluster-name", clusterName).Msg("use cluster name from node name")
				}
			}

			clusterName = "K8s Cluster " + clusterName
		}

		clusterAsset = &asset.Asset{
			PlatformIds: []string{clusterIdentifier},
			Name:        clusterName,
			Platform:    pf,
			Connections: []*providers.Config{tc}, // pass-in the current config
			State:       asset.State_STATE_RUNNING,
		}
		resolved = append(resolved, clusterAsset)

		if features.IsActive(cnquery.K8sNodeDiscovery) {
			// nodes are only added as related assets because we have no policies to scan them
			nodes, nodeRelationshipInfos, err := ListNodes(p, tc, clusterIdentifier)
			if err == nil && len(nodes) > 0 {
				ri := nodeRelationshipInfos[0]
				if ri.cloudAccountAsset != nil {
					clusterAsset.RelatedAssets = append(clusterAsset.RelatedAssets, ri.cloudAccountAsset)
				}
				clusterAsset.RelatedAssets = append(clusterAsset.RelatedAssets, nodes...)
			}
		}
	}

	additionalAssets, err := addSeparateAssets(tc, p, NamespaceFilterOpts{}, resourcesFilter, clusterIdentifier, ownershipDir, features)
	if err != nil {
		return nil, err
	}

	if clusterAsset != nil {
		isRelatedFn := func(a *asset.Asset) bool {
			return a.Platform.GetKind() == providers.Kind_KIND_K8S_OBJECT
		}

		for _, aa := range additionalAssets {
			if isRelatedFn(aa) {
				clusterAsset.RelatedAssets = append(clusterAsset.RelatedAssets, aa)
			}
		}
	}
	resolved = append(resolved, additionalAssets...)

	return resolved, nil
}

func (r *ClusterResolver) InitCtx(ctx context.Context) context.Context {
	return resources.SetDiscoveryCache(ctx, resources.NewDiscoveryCache())
}
