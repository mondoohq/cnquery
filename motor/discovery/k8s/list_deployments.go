package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/k8s"

	appsv1 "k8s.io/api/apps/v1"
)

// ListDeployments lits all deployments in the cluster.
func ListDeployments(transport k8s.KubernetesProvider, connection *providers.TransportConfig, clusterIdentifier string, namespaceFilter []string) ([]*asset.Asset, error) {
	namespaces, err := transport.Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "could not list kubernetes namespaces")
	}

	deployments := []appsv1.Deployment{}
	for i := range namespaces {
		namespace := namespaces[i]
		if !isIncluded(namespace.Name, namespaceFilter) {
			log.Info().Str("namespace", namespace.Name).Strs("filter", namespaceFilter).Msg("namespace not included")
			continue
		}

		deploymentsPerNamespace, err := transport.Deployments(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list deployments")
		}

		deployments = append(deployments, deploymentsPerNamespace...)
	}

	assets := []*asset.Asset{}
	for i := range deployments {
		deployment := deployments[i]
		platformData := transport.PlatformInfo()
		platformData.Version = deployment.APIVersion
		platformData.Build = deployment.ResourceVersion
		platformData.Labels = map[string]string{
			"namespace": deployment.Namespace,
			"uid":       string(deployment.UID),
		}
		platformData.Kind = providers.Kind_KIND_K8S_OBJECT
		asset := &asset.Asset{
			PlatformIds: []string{k8s.NewPlatformWorkloadId(clusterIdentifier, "deployments", deployment.Namespace, deployment.Name)},
			Name:        deployment.Namespace + "/" + deployment.Name,
			Platform:    platformData,
			Connections: []*providers.TransportConfig{connection},
			State:       asset.State_STATE_ONLINE,
			Labels:      deployment.Labels,
		}
		if asset.Labels == nil {
			asset.Labels = map[string]string{
				"namespace": deployment.Namespace,
			}
		} else {
			asset.Labels["namespace"] = deployment.Namespace
		}
		log.Debug().Str("name", deployment.Name).Str("connection", asset.Connections[0].Host).Msg("resolved deployment")

		assets = append(assets, asset)
	}

	return assets, nil
}
