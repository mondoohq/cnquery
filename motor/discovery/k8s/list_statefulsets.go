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
func ListStatefulSets(p k8s.KubernetesProvider, connection *providers.Config, clusterIdentifier string, namespaceFilter []string, od *k8s.PlatformIdOwnershipDirectory) ([]*asset.Asset, error) {
	namespaces, err := p.Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "could not list kubernetes namespaces")
	}

	statefulSets := []appsv1.StatefulSet{}
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

	assets := []*asset.Asset{}
	for i := range statefulSets {
		statefulSet := statefulSets[i]
		od.Add(&statefulSet)
		asset, err := createAssetFromObject(&statefulSet, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from statefulset")
		}

		log.Debug().Str("name", statefulSet.Name).Str("connection", asset.Connections[0].Host).Msg("resolved StatefulSet")

		assets = append(assets, asset)
	}

	return assets, nil
}
