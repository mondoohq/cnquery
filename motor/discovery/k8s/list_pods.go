package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/k8s"
	v1 "k8s.io/api/core/v1"
)

// ListPods list all pods in the cluster.
func ListPods(transport k8s.Transport, connection *providers.TransportConfig, clusterIdentifier string, namespaceFilter []string) ([]*asset.Asset, error) {
	namespaces, err := transport.Namespaces()
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

		podsPerNamespace, err := transport.Pods(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list pods")
		}

		pods = append(pods, podsPerNamespace...)
	}

	assets := []*asset.Asset{}
	for i := range pods {
		pod := pods[i]
		platformData := transport.PlatformInfo()
		platformData.Version = pod.APIVersion
		platformData.Build = pod.ResourceVersion
		platformData.Labels = map[string]string{
			"namespace": pod.Namespace,
			"uid":       string(pod.UID),
		}
		platformData.Kind = providers.Kind_KIND_K8S_OBJECT
		asset := &asset.Asset{
			PlatformIds: []string{k8s.NewPlatformWorkloadId(clusterIdentifier, "pods", pod.Namespace, pod.Name)},
			Name:        pod.Namespace + "/" + pod.Name,
			Platform:    platformData,
			Connections: []*providers.TransportConfig{connection},
			State:       asset.State_STATE_ONLINE,
			Labels:      pod.Labels,
		}
		log.Debug().Str("name", pod.Name).Str("connection", asset.Connections[0].Host).Msg("resolved pod")

		assets = append(assets, asset)
	}

	return assets, nil
}
