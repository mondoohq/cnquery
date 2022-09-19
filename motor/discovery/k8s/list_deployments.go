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
func ListDeployments(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter []string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	deployments := []appsv1.Deployment{}

	if len(resFilter) > 0 {
		// If there is a resources filter we should only retrieve the deployments that are in the filter.
		if len(resFilter["deployment"]) == 0 {
			return []*asset.Asset{}, nil
		}

		for _, res := range resFilter["deployment"] {
			deployment, err := p.Deployment(res.Namespace, res.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get deployment %s/%s", res.Namespace, res.Name)
			}

			deployments = append(deployments, *deployment)
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

			deploymentsPerNamespace, err := p.Deployments(namespace)
			if err != nil {
				return nil, errors.Wrap(err, "failed to list deployments")
			}

			deployments = append(deployments, deploymentsPerNamespace...)
		}
	}

	assets := []*asset.Asset{}
	for i := range deployments {
		deployment := deployments[i]
		od.Add(&deployment)

		asset, err := createAssetFromObject(&deployment, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from deployment")
		}

		log.Debug().Str("name", deployment.Name).Str("connection", asset.Connections[0].Host).Msg("resolved deployment")

		assets = append(assets, asset)
	}

	return assets, nil
}
