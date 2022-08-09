package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/k8s"
	appsv1 "k8s.io/api/apps/v1"
)

// ListReplicaSets list all replicaSets in the cluster.
func ListReplicaSets(transport k8s.Transport, connection *providers.TransportConfig, clusterIdentifier string, namespaceFilter []string) ([]*asset.Asset, error) {
	namespaces, err := transport.Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "could not list kubernetes namespaces")
	}

	replicaSets := []appsv1.ReplicaSet{}
	for i := range namespaces {
		namespace := namespaces[i]
		if !isIncluded(namespace.Name, namespaceFilter) {
			log.Info().Str("namespace", namespace.Name).Strs("filter", namespaceFilter).Msg("namespace not included")
			continue
		}

		replicaSetsPerNamespace, err := transport.ReplicaSets(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list replicasets")
		}

		replicaSets = append(replicaSets, replicaSetsPerNamespace...)
	}

	assets := []*asset.Asset{}
	for i := range replicaSets {
		replicaSet := replicaSets[i]
		platformData := transport.PlatformInfo()
		platformData.Version = replicaSet.APIVersion
		platformData.Build = replicaSet.ResourceVersion
		platformData.Labels = map[string]string{
			"namespace": replicaSet.Namespace,
			"uid":       string(replicaSet.UID),
		}
		platformData.Kind = providers.Kind_KIND_K8S_OBJECT
		asset := &asset.Asset{
			PlatformIds: []string{k8s.NewPlatformWorkloadId(clusterIdentifier, "replicasets", replicaSet.Namespace, replicaSet.Name)},
			Name:        replicaSet.Namespace + "/" + replicaSet.Name,
			Platform:    platformData,
			Connections: []*providers.TransportConfig{connection},
			State:       asset.State_STATE_ONLINE,
			Labels:      replicaSet.Labels,
		}
		log.Debug().Str("name", replicaSet.Name).Str("connection", asset.Connections[0].Host).Msg("resolved replicaset")

		assets = append(assets, asset)
	}

	return assets, nil
}
