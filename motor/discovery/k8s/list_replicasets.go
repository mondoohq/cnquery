package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	appsv1 "k8s.io/api/apps/v1"
)

// ListReplicaSets list all replicaSets in the cluster.
func ListReplicaSets(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter []string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	replicaSets := []appsv1.ReplicaSet{}

	if len(resFilter) > 0 {
		// If there is a resources filter we should only retrieve the replicasets that are in the filter.
		if len(resFilter["replicaset"]) == 0 {
			return []*asset.Asset{}, nil
		}

		for _, res := range resFilter["replicaset"] {
			rs, err := p.ReplicaSet(res.Namespace, res.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get replicaset %s/%s", res.Namespace, res.Name)
			}

			replicaSets = append(replicaSets, *rs)
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

			replicaSetsPerNamespace, err := p.ReplicaSets(namespace)
			if err != nil {
				return nil, errors.Wrap(err, "failed to list replicasets")
			}

			replicaSets = append(replicaSets, replicaSetsPerNamespace...)
		}
	}

	assets := []*asset.Asset{}
	for i := range replicaSets {
		replicaSet := replicaSets[i]
		if od != nil {
			od.Add(&replicaSet)
		}
		asset, err := createAssetFromObject(&replicaSet, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from repicaset")
		}

		log.Debug().Str("name", replicaSet.Name).Str("connection", asset.Connections[0].Host).Msg("resolved replicaset")

		assets = append(assets, asset)
	}

	return assets, nil
}
