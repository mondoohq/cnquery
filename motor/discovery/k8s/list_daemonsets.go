package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/k8s"

	appsv1 "k8s.io/api/apps/v1"
)

// ListDaemonSets list all daemonsets in the cluster.
func ListDaemonSets(transport k8s.Transport, connection *providers.TransportConfig, clusterIdentifier string, namespaceFilter []string) ([]*asset.Asset, error) {
	namespaces, err := transport.Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "could not list kubernetes namespaces")
	}

	daemonsets := []appsv1.DaemonSet{}
	for i := range namespaces {
		namespace := namespaces[i]
		if !isIncluded(namespace.Name, namespaceFilter) {
			log.Info().Str("namespace", namespace.Name).Strs("filter", namespaceFilter).Msg("namespace not included")
			continue
		}

		daemonsetsPerNamespace, err := transport.DaemonSets(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list daemonsets")
		}

		daemonsets = append(daemonsets, daemonsetsPerNamespace...)
	}

	assets := []*asset.Asset{}
	for i := range daemonsets {
		daemonset := daemonsets[i]
		podPlatform := transport.PlatformInfo()
		podPlatform.Version = daemonset.APIVersion
		podPlatform.Build = daemonset.ResourceVersion
		podPlatform.Labels = map[string]string{
			"namespace": daemonset.Namespace,
			"uid":       string(daemonset.UID),
		}
		podPlatform.Kind = providers.Kind_KIND_K8S_OBJECT
		asset := &asset.Asset{
			PlatformIds: []string{k8s.NewPlatformWorkloadId(clusterIdentifier, "daemonsets", daemonset.Namespace, daemonset.Name)},
			Name:        daemonset.Namespace + "/" + daemonset.Name,
			Platform:    podPlatform,
			Connections: []*providers.TransportConfig{connection},
			State:       asset.State_STATE_ONLINE,
			Labels:      daemonset.Labels,
		}
		if asset.Labels == nil {
			asset.Labels = map[string]string{
				"namespace": daemonset.Namespace,
			}
		} else {
			asset.Labels["namespace"] = daemonset.Namespace
		}
		log.Debug().Str("name", daemonset.Name).Str("connection", asset.Connections[0].Host).Msg("resolved daemonset")

		assets = append(assets, asset)
	}

	return assets, nil
}
