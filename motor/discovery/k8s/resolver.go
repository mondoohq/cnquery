package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/resources/packs/os/kubectl"
)

var _ common.ContextInitializer = (*Resolver)(nil)

const (
	DiscoveryAll              = "all"
	DiscoveryPods             = "pods"
	DiscoveryJobs             = "jobs"
	DiscoveryCronJobs         = "cronjobs"
	DiscoveryStatefulSets     = "statefulsets"
	DiscoveryDeployments      = "deployments"
	DiscoveryReplicaSets      = "replicasets"
	DiscoveryDaemonSets       = "daemonsets"
	DiscoveryContainerImages  = "container-images"
	DiscoveryAdmissionReviews = "admissionreviews"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Kubernetes Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{
		DiscoveryAll,
		DiscoveryPods,
		DiscoveryJobs,
		DiscoveryCronJobs,
		DiscoveryStatefulSets,
		DiscoveryDeployments,
		DiscoveryReplicaSets,
		DiscoveryDaemonSets,
		DiscoveryContainerImages,
		DiscoveryAdmissionReviews,
	}
}

type K8sResourceIdentifier struct {
	Type      string
	Namespace string
	Name      string
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}
	namespacesFilter := []string{}

	var k8sctlConfig *kubectl.KubectlConfig
	localProvider, err := local.New()
	if err == nil {
		k8sctlConfig, err = kubectl.LoadKubeConfig(localProvider)
		if err != nil {
			return nil, err
		}
	}

	allNamespaces := tc.Options["all-namespaces"]
	if allNamespaces != "true" {
		namespace := tc.Options["namespace"]
		if len(namespace) > 0 {
			namespacesFilter = append(namespacesFilter, namespace)
		} else {
			// try parse the current kubectl namespace
			if k8sctlConfig != nil && len(k8sctlConfig.CurrentNamespace()) > 0 {
				namespacesFilter = append(namespacesFilter, k8sctlConfig.CurrentNamespace())
			}
		}
	}

	log.Debug().Strs("namespaceFilter", namespacesFilter).Msg("resolve k8s assets")

	p, err := k8s.New(ctx, tc)
	if err != nil {
		return nil, err
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

		clusterName = "K8S Cluster " + clusterName
		ns, ok := tc.Options[k8s.OPTION_NAMESPACE]
		if ok && ns != "" {
			clusterName += " (Namespace: " + ns + ")"
		}
	}

	resourcesFilter, err := resourceFilters(tc)
	if err != nil {
		return nil, err
	}

	// Only discover cluster and nodes if there are no resource filters. For CI/CD do not
	// discover the cluster asset at all. In that case that would be the admission review resource
	// for which we only care if we have explictly enabled discovery for it.
	var clusterAsset *asset.Asset
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	if len(resourcesFilter) == 0 && root.Category != asset.AssetCategory_CATEGORY_CICD {
		clusterAsset = &asset.Asset{
			PlatformIds: []string{clusterIdentifier},
			Name:        clusterName,
			Platform:    pf,
			Connections: []*providers.Config{tc}, // pass-in the current config
			State:       asset.State_STATE_RUNNING,
		}
		resolved = append(resolved, clusterAsset)

		// nodes are only added as related assets because we have no policies to scan them
		nodes, nodeRelationshipInfos, err := ListNodes(p, tc, clusterIdentifier, namespacesFilter)
		if err == nil && len(nodes) > 0 {
			ri := nodeRelationshipInfos[0]
			if ri.cloudAccountAsset != nil {
				clusterAsset.RelatedAssets = append(clusterAsset.RelatedAssets, ri.cloudAccountAsset)
			}
			clusterAsset.RelatedAssets = append(clusterAsset.RelatedAssets, nodes...)
		}
	}

	additionalAssets, err := addSeparateAssets(tc, p, namespacesFilter, resourcesFilter, clusterIdentifier, ownershipDir)
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
	namespacesFilter []string,
	resourcesFilter map[string][]K8sResourceIdentifier,
	clusterIdentifier string,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// discover deployments
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryDeployments) {
		// fetch deployment information
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for deployments")
		connection := tc.Clone()
		deployments, err := ListDeployments(p, connection, clusterIdentifier, namespacesFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s deployments")
			return nil, err
		}
		resolved = append(resolved, deployments...)
	}

	// discover k8s pods
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryPods) {
		// fetch pod information
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for pods")
		connection := tc.Clone()
		pods, err := ListPods(p, connection, clusterIdentifier, namespacesFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s pods")
			return nil, err
		}
		resolved = append(resolved, pods...)
	}

	// discovery k8s daemonsets
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryDaemonSets) {
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for daemonsets")
		connection := tc.Clone()
		daemonsets, err := ListDaemonSets(p, connection, clusterIdentifier, namespacesFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s daemonsets")
			return nil, err
		}
		resolved = append(resolved, daemonsets...)
	}

	// discover k8s pod images
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryContainerImages) {
		// fetch pod information
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for pods images")
		containerimages, err := ListPodImages(p, namespacesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s pods images")
			return nil, err
		}
		resolved = append(resolved, containerimages...)
	}

	// discover cronjobs
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryCronJobs) {
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for cronjobs")
		connection := tc.Clone()
		cronjobs, err := ListCronJobs(p, connection, clusterIdentifier, namespacesFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s cronjobs")
			return nil, err
		}
		resolved = append(resolved, cronjobs...)
	}

	// discover statefulsets
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryStatefulSets) {
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for statefulsets")
		connection := tc.Clone()
		statefulsets, err := ListStatefulSets(p, connection, clusterIdentifier, namespacesFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s statefulsets")
			return nil, err
		}
		resolved = append(resolved, statefulsets...)
	}

	// discover jobs
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryJobs) {
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for jobs")
		connection := tc.Clone()
		jobs, err := ListJobs(p, connection, clusterIdentifier, namespacesFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s jobs")
			return nil, err
		}
		resolved = append(resolved, jobs...)
	}

	// discover replicasets
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryReplicaSets) {
		log.Debug().Strs("namespace", namespacesFilter).Msg("search for replicasets")
		connection := tc.Clone()
		replicasets, err := ListReplicaSets(p, connection, clusterIdentifier, namespacesFilter, resourcesFilter, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s replicasets")
			return nil, err
		}
		resolved = append(resolved, replicasets...)
	}

	// discover admissionreviews
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryAdmissionReviews) {
		log.Debug().Msg("search for admissionreviews")
		connection := tc.Clone()
		admissionReviews, err := ListAdmissionReviews(p, connection, clusterIdentifier, od)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch k8s admissionreviews")
			return nil, err
		}
		resolved = append(resolved, admissionReviews...)
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
					if err != nil {
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
