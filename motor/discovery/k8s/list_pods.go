package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	v1 "k8s.io/api/core/v1"
)

// ListPods list all pods in the cluster.
func ListPods(p k8s.KubernetesProvider, connection *providers.Config, clusterIdentifier string, namespaceFilter []string) ([]*asset.Asset, error) {
	namespaces, err := p.Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "could not list kubernetes namespaces")
	}

	pods := []v1.Pod{}
	for i := range namespaces {
		namespace := namespaces[i]
		if !isIncluded(namespace.Name, namespaceFilter) {
			log.Info().Str("namespace", namespace.Name).Strs("filter", namespaceFilter).Msg("namespace not included")
			continue
		}

		podsPerNamespace, err := p.Pods(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list pods")
		}

		pods = append(pods, podsPerNamespace...)
	}

	assets := []*asset.Asset{}
	for i := range pods {
		pod := pods[i]
		asset, err := createAssetFromObject(&pod, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from pod")
		}

		log.Debug().Str("name", pod.Name).Str("connection", asset.Connections[0].Host).Msg("resolved pod")

		assets = append(assets, asset)
	}

	return assets, nil
}
