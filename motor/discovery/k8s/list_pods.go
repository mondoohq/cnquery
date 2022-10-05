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
func ListPods(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter []string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	pods := []v1.Pod{}

	if len(resFilter) > 0 {
		// If there is a resources filter we should only retrieve the pods that are in the filter.
		if len(resFilter["pod"]) == 0 {
			return []*asset.Asset{}, nil
		}

		for _, res := range resFilter["pod"] {
			p, err := p.Pod(res.Namespace, res.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get pod %s/%s", res.Namespace, res.Name)
			}

			pods = append(pods, *p)
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

			podsPerNamespace, err := p.Pods(namespace)
			if err != nil {
				return nil, errors.Wrap(err, "failed to list pods")
			}

			pods = append(pods, podsPerNamespace...)
		}
	}

	assets := []*asset.Asset{}
	for i := range pods {
		pod := pods[i]
		od.Add(&pod)

		asset, err := createAssetFromObject(&pod, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from pod")
		}

		log.Debug().Str("name", pod.Name).Str("connection", asset.Connections[0].Host).Msg("resolved pod")

		assets = append(assets, asset)
	}

	return assets, nil
}
