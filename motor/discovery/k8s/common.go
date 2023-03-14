package k8s

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
)

type K8sResourceIdentifier struct {
	Type      string
	Namespace string
	Name      string
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
