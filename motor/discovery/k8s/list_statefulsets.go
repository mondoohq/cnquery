package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/k8s"

	appsv1 "k8s.io/api/apps/v1"
)

// ListStatefulSets list all statefulsets in the cluster.
func ListStatefulSets(transport k8s.KubernetesProvider, connection *providers.TransportConfig, clusterIdentifier string, namespaceFilter []string) ([]*asset.Asset, error) {
	namespaces, err := transport.Namespaces()
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

		statefulSetsPerNamespace, err := transport.StatefulSets(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list StatefulSets")
		}

		statefulSets = append(statefulSets, statefulSetsPerNamespace...)
	}

	assets := []*asset.Asset{}
	for i := range statefulSets {
		statefulSet := statefulSets[i]
		platformData := transport.PlatformInfo()
		platformData.Version = statefulSet.APIVersion
		platformData.Build = statefulSet.ResourceVersion
		platformData.Labels = map[string]string{
			"namespace": statefulSet.Namespace,
			"uid":       string(statefulSet.UID),
		}
		platformData.Kind = providers.Kind_KIND_K8S_OBJECT
		asset := &asset.Asset{
			PlatformIds: []string{k8s.NewPlatformWorkloadId(clusterIdentifier, "statefulsets", statefulSet.Namespace, statefulSet.Name)},
			Name:        statefulSet.Namespace + "/" + statefulSet.Name,
			Platform:    platformData,
			Connections: []*providers.TransportConfig{connection},
			State:       asset.State_STATE_ONLINE,
			Labels:      statefulSet.Labels,
		}
		if asset.Labels == nil {
			asset.Labels = map[string]string{
				"namespace": statefulSet.Namespace,
			}
		} else {
			asset.Labels["namespace"] = statefulSet.Namespace
		}
		log.Debug().Str("name", statefulSet.Name).Str("connection", asset.Connections[0].Host).Msg("resolved StatefulSet")

		assets = append(assets, asset)
	}

	return assets, nil
}
