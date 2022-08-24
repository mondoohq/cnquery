package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"

	appsv1 "k8s.io/api/apps/v1"
)

// ListDeployments lits all deployments in the cluster.
func ListDeployments(p k8s.KubernetesProvider, connection *providers.Config, clusterIdentifier string, namespaceFilter []string) ([]*asset.Asset, error) {
	namespaces, err := p.Namespaces()
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

		deploymentsPerNamespace, err := p.Deployments(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list deployments")
		}

		deployments = append(deployments, deploymentsPerNamespace...)
	}

	assets := []*asset.Asset{}
	for i := range deployments {
		deployment := deployments[i]
		asset, err := createAssetFromObject(&deployment, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from deployment")
		}

		log.Debug().Str("name", deployment.Name).Str("connection", asset.Connections[0].Host).Msg("resolved deployment")

		assets = append(assets, asset)
	}

	return assets, nil
}
