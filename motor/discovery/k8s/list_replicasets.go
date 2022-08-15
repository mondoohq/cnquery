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
func ListReplicaSets(p k8s.KubernetesProvider, connection *providers.Config, clusterIdentifier string, namespaceFilter []string) ([]*asset.Asset, error) {
	namespaces, err := p.Namespaces()
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

		replicaSetsPerNamespace, err := p.ReplicaSets(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list replicasets")
		}

		replicaSets = append(replicaSets, replicaSetsPerNamespace...)
	}

	assets := []*asset.Asset{}
	for i := range replicaSets {
		replicaSet := replicaSets[i]
		platformData := p.PlatformInfo()
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
			Connections: []*providers.Config{connection},
			State:       asset.State_STATE_ONLINE,
			Labels:      replicaSet.Labels,
		}
		if asset.Labels == nil {
			asset.Labels = map[string]string{
				"namespace": replicaSet.Namespace,
			}
		} else {
			asset.Labels["namespace"] = replicaSet.Namespace
		}
		log.Debug().Str("name", replicaSet.Name).Str("connection", asset.Connections[0].Host).Msg("resolved replicaset")

		assets = append(assets, asset)
	}

	return assets, nil
}
