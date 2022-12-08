package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/meta"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
)

// ListIngresses lists all ingresses in the cluster.
func ListIngresses(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	nsFilter NamespaceFilterOpts,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	ingresses := []*networkingv1.Ingress{}

	if len(resFilter) > 0 {
		if len(resFilter["ingress"]) == 0 {
			return []*asset.Asset{}, nil
		}

		for _, res := range resFilter["ingress"] {
			ingress, err := p.Ingress(res.Namespace, res.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get ingress %s/%s", res.Namespace, res.Name)
			}

			ingresses = append(ingresses, ingress)
		}

	} else {
		namespaces, err := p.Namespaces()
		if err != nil {
			return nil, errors.Wrap(err, "could not list kubernetes namespaces")
		}

		for _, ns := range namespaces {
			skip, err := skipNamespace(ns, nsFilter)
			if err != nil {
				log.Error().Err(err).Str("namespace", ns.Name).Msg("error checking whether Namespace should be included or excluded")
				return nil, err
			}
			if skip {
				log.Debug().Str("namespace", ns.Name).Msg("ignoring namespace")
				continue
			}

			ingressesPerNamespace, err := p.Ingresses(ns)
			if err != nil {
				return nil, errors.Wrap(err, "failed to list ingresses")
			}
			ingresses = append(ingresses, ingressesPerNamespace...)
		}
	}

	assets := []*asset.Asset{}
	for i := range ingresses {
		od.Add(ingresses[i])

		asset, err := createAssetFromObject(ingresses[i], p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from ingress")
		}

		obj, _ := meta.Accessor(ingresses[i])
		log.Debug().Str("name", obj.GetName()).Str("connection", asset.Connections[0].Host).Msg("resolved ingress")

		assets = append(assets, asset)
	}
	return assets, nil
}
