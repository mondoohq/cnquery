package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"

	appsv1 "k8s.io/api/apps/v1"
)

// ListDaemonSets list all daemonsets in the cluster.
func ListDaemonSets(p k8s.KubernetesProvider, connection *providers.Config, clusterIdentifier string, namespaceFilter []string) ([]*asset.Asset, error) {
	namespaces, err := p.Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "could not list kubernetes namespaces")
	}

	daemonSets := []appsv1.DaemonSet{}
	for i := range namespaces {
		namespace := namespaces[i]
		if !isIncluded(namespace.Name, namespaceFilter) {
			log.Info().Str("namespace", namespace.Name).Strs("filter", namespaceFilter).Msg("namespace not included")
			continue
		}

		daemonSetsPerNamespace, err := p.DaemonSets(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list daemonsets")
		}

		daemonSets = append(daemonSets, daemonSetsPerNamespace...)
	}

	assets := []*asset.Asset{}
	for i := range daemonSets {
		daemonSet := daemonSets[i]
		asset, err := createAssetFromObject(&daemonSet, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from daemonset")
		}

		log.Debug().Str("name", daemonSet.Name).Str("connection", asset.Connections[0].Host).Msg("resolved daemonset")

		assets = append(assets, asset)
	}

	return assets, nil
}
