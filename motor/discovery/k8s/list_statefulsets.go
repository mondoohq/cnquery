package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"

	appsv1 "k8s.io/api/apps/v1"
)

// ListStatefulSets list all statefulsets in the cluster.
func ListStatefulSets(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter []string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	statefulSets := []appsv1.StatefulSet{}

	if len(resFilter) > 0 {
		// If there is a resources filter we should only retrieve the statefulsets that are in the filter.
		if len(resFilter["statefulset"]) == 0 {
			return []*asset.Asset{}, nil
		}

		for _, res := range resFilter["statefulset"] {
			ss, err := p.StatefulSet(res.Namespace, res.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get statefulset %s/%s", res.Namespace, res.Name)
			}

			statefulSets = append(statefulSets, *ss)
		}
	} else {
		namespaces, err := p.Namespaces()
		if err != nil {
			return nil, errors.Wrap(err, "could not list kubernetes namespaces")
		}

		for i := range namespaces {
			namespace := namespaces[i]
			if !isIncluded(namespace.Name, namespaceFilter) {
				log.Info().Str("namespace", namespace.Name).Strs("filter", namespaceFilter).Msg("namespace not included")
				continue
			}

			statefulSetsPerNamespace, err := p.StatefulSets(namespace)
			if err != nil {
				return nil, errors.Wrap(err, "failed to list StatefulSets")
			}

			statefulSets = append(statefulSets, statefulSetsPerNamespace...)
		}
	}

	assets := []*asset.Asset{}
	for i := range statefulSets {
		statefulSet := statefulSets[i]
		if od != nil {
			od.Add(&statefulSet)
		}
		asset, err := createAssetFromObject(&statefulSet, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from statefulset")
		}

		log.Debug().Str("name", statefulSet.Name).Str("connection", asset.Connections[0].Host).Msg("resolved StatefulSet")

		assets = append(assets, asset)
	}

	return assets, nil
}
