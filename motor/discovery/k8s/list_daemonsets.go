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
func ListDaemonSets(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter []string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	daemonSets := []appsv1.DaemonSet{}

	if len(resFilter) > 0 {
		// If there is a resources filter we should only retrieve the daemonsets that are in the filter.
		if len(resFilter["daemonset"]) == 0 {
			return []*asset.Asset{}, nil
		}

		for _, res := range resFilter["daemonset"] {
			ds, err := p.DaemonSet(res.Namespace, res.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get daemonset %s/%s", res.Namespace, res.Name)
			}

			daemonSets = append(daemonSets, *ds)
		}
	} else {
		namespaces, err := p.Namespaces()
		if err != nil {
			return nil, errors.Wrap(err, "could not list kubernetes namespaces")
		}

		for i := range namespaces {
			namespace := namespaces[i]
			if !isIncluded(namespace.Name, namespaceFilter) {
				log.Debug().Str("namespace", namespace.Name).Strs("filter", namespaceFilter).Msg("namespace not included")
				continue
			}

			daemonSetsPerNamespace, err := p.DaemonSets(namespace)
			if err != nil {
				return nil, errors.Wrap(err, "failed to list daemonsets")
			}

			daemonSets = append(daemonSets, daemonSetsPerNamespace...)
		}
	}

	assets := []*asset.Asset{}
	for i := range daemonSets {
		daemonSet := daemonSets[i]
		od.Add(&daemonSet)

		asset, err := createAssetFromObject(&daemonSet, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from daemonset")
		}

		log.Debug().Str("name", daemonSet.Name).Str("connection", asset.Connections[0].Host).Msg("resolved daemonset")

		assets = append(assets, asset)
	}

	return assets, nil
}
