// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
)

// ListNamespaces lists all namespaces in the cluster.
func ListNamespaces(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	nsFilter NamespaceFilterOpts,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	nss := []*v1.Namespace{}

	if len(resFilter) > 0 {
		if len(resFilter["namespace"]) == 0 {
			return []*asset.Asset{}, nil
		}

		for _, res := range resFilter["namespace"] {
			ns, err := p.Namespace(res.Name)
			if err != nil {
				return nil, err
			}
			nss = append(nss, ns)
		}
	} else {
		namespaces, err := p.Namespaces()
		if err != nil {
			// If we don't have rights to list the cluster namespaces, attempt getting them 1 by 1
			if k8sErrors.IsForbidden(err) && len(nsFilter.include) > 0 {
				for _, ns := range nsFilter.include {
					n, err := p.Namespace(ns)
					if err != nil {
						return nil, err
					}
					nss = append(nss, n)
				}
			} else {
				return nil, errors.Wrap(err, "could not list kubernetes namespaces")
			}
		}

		for i := range namespaces {
			ns := namespaces[i]
			skip, err := skipNamespace(ns, nsFilter)
			if err != nil {
				log.Error().Err(err).Str("namespace", ns.Name).Msg("error checking whether Namespace should be included or excluded")
				return nil, err
			}
			if skip {
				log.Debug().Str("namespace", ns.Name).Msg("ignoring namespace")
				continue
			}
			nss = append(nss, &ns)
		}
	}

	assets := []*asset.Asset{}
	for i := range nss {
		// The namespace can be a root asset, in this case the ownership directory will not be present
		if od != nil {
			od.Add(nss[i])
		}

		asset, err := createAssetFromObject(nss[i], p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from namespace")
		}

		obj, _ := meta.Accessor(nss[i])
		log.Debug().Str("name", obj.GetName()).Str("connection", asset.Connections[0].Host).Msg("resolved namespace")

		assets = append(assets, asset)
	}
	return assets, nil
}
