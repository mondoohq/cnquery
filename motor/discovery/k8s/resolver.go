package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/gobwas/glob"
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
	"k8s.io/apimachinery/pkg/api/errors"
)

var _ common.ContextInitializer = (*Resolver)(nil)

const (
	DiscoveryClusters         = "clusters"
	DiscoveryPods             = "pods"
	DiscoveryJobs             = "jobs"
	DiscoveryCronJobs         = "cronjobs"
	DiscoveryStatefulSets     = "statefulsets"
	DiscoveryDeployments      = "deployments"
	DiscoveryReplicaSets      = "replicasets"
	DiscoveryDaemonSets       = "daemonsets"
	DiscoveryContainerImages  = "container-images"
	DiscoveryAdmissionReviews = "admissionreviews"
	DiscoveryIngresses        = "ingresses"
)

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
	}
}

type K8sResourceIdentifier struct {
	Type      string
	Namespace string
	Name      string
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	features := cnquery.GetFeatures(ctx)
	resolved := []*asset.Asset{}
	nsFilter := NamespaceFilterOpts{
		include: []string{},
	}

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

	// if --namespace and --namespaces were both specified, just combine them into a single
	// list of Namespaces to allow resources from

	// FIXME: DEPRECATED, remove in v8.0 vv
	namespaceOpt := tc.Options["namespace"]
	if len(namespaceOpt) > 0 {
		log.Info().Msgf("namespace filter has been set to %q", namespaceOpt)
		nsFilter.include = append(nsFilter.include, namespaceOpt)

	}
	// ^^

	includeNamespaces := tc.Options["namespaces"]
	if len(includeNamespaces) > 0 {
		nsFilter.include = append(nsFilter.include, strings.Split(includeNamespaces, ",")...)
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

	resourcesFilter, err := resourceFilters(tc)
	if err != nil {
		return nil, err
	}

	if tc.IncludesDiscoveryTarget(common.DiscoveryAuto) {
		log.Info().Msg("discovery option auto is used. This will detect the assets: cluster, jobs, cronjobs, pods, statefulsets, deployments, replicasets, daemonsets")
	}

	// Only discover cluster and nodes if there are no resource filters.
	var clusterAsset *asset.Asset
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryClusters) &&
		len(resourcesFilter) == 0 {
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

	additionalAssets, err := addSeparateAssets(tc, p, nsFilter, resourcesFilter, clusterIdentifier, ownershipDir, features)
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

func (r *Resolver) InitCtx(ctx context.Context) context.Context {
	return resources.SetDiscoveryCache(ctx, resources.NewDiscoveryCache())
}

// addSeparateAssets Depending on config options it will search for additional assets which should be listed separately.
func addSeparateAssets(
	tc *providers.Config,
	p k8s.KubernetesProvider,
	nsFilter NamespaceFilterOpts,
	resourcesFilter map[string][]K8sResourceIdentifier,
	clusterIdentifier string,
	od *k8s.PlatformIdOwnershipDirectory,
	features cnquery.Features,
) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// discover deployments
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryDeployments) {
		// fetch deployment information
		log.Debug().Strs("namespace", nsFilter.include).Msg("search for deployments")
		connection := tc.Clone()
		deployments, err := ListDeployments(p, connection, clusterIdentifier, nsFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s deployments")
			return nil, err
		}
		resolved = append(resolved, deployments...)
	}

	// discover k8s pods
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryPods) {
		// fetch pod information
		log.Debug().Strs("namespace", nsFilter.include).Msg("search for pods")
		connection := tc.Clone()
		pods, err := ListPods(p, connection, clusterIdentifier, nsFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s pods")
			return nil, err
		}
		resolved = append(resolved, pods...)
	}

	// discover k8s pod images
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryContainerImages) {
		// fetch pod information
		log.Debug().Strs("namespace", nsFilter.include).Msg("search for pods images")
		containerimages, err := ListPodImages(p, nsFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s pods images")
			return nil, err
		}
		resolved = append(resolved, containerimages...)
	}

	// discovery k8s daemonsets
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryDaemonSets) {
		log.Debug().Strs("namespace", nsFilter.include).Msg("search for daemonsets")
		connection := tc.Clone()
		daemonsets, err := ListDaemonSets(p, connection, clusterIdentifier, nsFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s daemonsets")
			return nil, err
		}
		resolved = append(resolved, daemonsets...)
	}

	// discover cronjobs
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryCronJobs) {
		log.Debug().Strs("namespace", nsFilter.include).Msg("search for cronjobs")
		connection := tc.Clone()
		cronjobs, err := ListCronJobs(p, connection, clusterIdentifier, nsFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s cronjobs")
			return nil, err
		}
		resolved = append(resolved, cronjobs...)
	}

	// discover jobs
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryJobs, DiscoveryJobs) {
		log.Debug().Strs("namespace", nsFilter.include).Msg("search for jobs")
		connection := tc.Clone()
		jobs, err := ListJobs(p, connection, clusterIdentifier, nsFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s jobs")
			return nil, err
		}
		resolved = append(resolved, jobs...)
	}

	// discover statefulsets
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryStatefulSets) {
		log.Debug().Strs("namespace", nsFilter.include).Msg("search for statefulsets")
		connection := tc.Clone()
		statefulsets, err := ListStatefulSets(p, connection, clusterIdentifier, nsFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s statefulsets")
			return nil, err
		}
		resolved = append(resolved, statefulsets...)
	}

	// discover replicasets
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryReplicaSets) {
		log.Debug().Strs("namespace", nsFilter.include).Msg("search for replicasets")
		connection := tc.Clone()
		replicasets, err := ListReplicaSets(p, connection, clusterIdentifier, nsFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s replicasets")
			return nil, err
		}
		resolved = append(resolved, replicasets...)
	}

	// discover admissionreviews
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryAdmissionReviews) {
		log.Debug().Msg("search for admissionreviews")
		connection := tc.Clone()
		admissionReviews, err := ListAdmissionReviews(p, connection, clusterIdentifier, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s admissionreviews")
			return nil, err
		}
		resolved = append(resolved, admissionReviews...)
	}

	// discover ingresses
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryIngresses) {
		log.Debug().Msg("search for ingresses")
		connection := tc.Clone()
		ingresses, err := ListIngresses(p, connection, clusterIdentifier, nsFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s ingresses")
		}
		resolved = append(resolved, ingresses...)
	}

	// build a lookup on the k8s uid to look up individual assets to link
	platformIdToAssetMap := map[string]*asset.Asset{}
	for _, assetObj := range resolved {
		for _, platformId := range assetObj.PlatformIds {
			platformIdToAssetMap[platformId] = assetObj
		}
	}

	for id, a := range platformIdToAssetMap {
		ownedBy := od.OwnedBy(id)
		for _, ownerPlatformId := range ownedBy {
			if aa, ok := platformIdToAssetMap[ownerPlatformId]; ok {
				a.RelatedAssets = append(a.RelatedAssets, aa)
			} else {
				// If the owner object is not scanned we can still add an asset as we know most of the information
				// from the ownerReference field
				if platformEntry, ok := od.GetKubernetesObjectData(ownerPlatformId); ok {
					platformData, err := createPlatformData(platformEntry.Kind, providers.RUNTIME_KUBERNETES_CLUSTER)
					if err != nil || (!features.IsActive(cnquery.K8sNodeDiscovery) && platformData.Name == "k8s-node") {
						continue
					}
					a.RelatedAssets = append(a.RelatedAssets, &asset.Asset{
						PlatformIds: []string{ownerPlatformId},
						Platform:    platformData,
						Name:        platformEntry.Namespace + "/" + platformEntry.Name,
					})
				}
			}
		}
	}
	return resolved, nil
}

// resourceFilters parses the resource filters from the provider config
func resourceFilters(tc *providers.Config) (map[string][]K8sResourceIdentifier, error) {
	resourcesFilter := make(map[string][]K8sResourceIdentifier)
	if fOpt, ok := tc.Options["k8s-resources"]; ok {
		fs := strings.Split(fOpt, ",")
		for _, f := range fs {
			ids := strings.Split(strings.TrimSpace(f), ":")
			resType := ids[0]
			var ns, name string
			if _, ok := resourcesFilter[resType]; !ok {
				resourcesFilter[resType] = []K8sResourceIdentifier{}
			}

			switch len(ids) {
			case 3:
				// Namespaced resources have the format type:ns:name
				ns = ids[1]
				name = ids[2]
			case 2:
				// Non-namespaced resources have the format type:name
				name = ids[1]
			default:
				return nil, fmt.Errorf("invalid k8s resource filter: %s", f)
			}

			resourcesFilter[resType] = append(resourcesFilter[resType], K8sResourceIdentifier{Type: resType, Namespace: ns, Name: name})
		}
	}
	return resourcesFilter, nil
}
