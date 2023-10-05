// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package k8s

import (
	"fmt"

	"github.com/gobwas/glob"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/motor/asset"
	"go.mondoo.com/cnquery/v9/motor/providers"
	"go.mondoo.com/cnquery/v9/motor/providers/k8s"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

type NamespaceFilterOpts struct {
	include []string
	exclude []string
}

// ListCronJobs list all cronjobs in the cluster.
func ListCronJobs(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	nsFilter NamespaceFilterOpts,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListNamespacedObj(p, connection, clusterIdentifier, nsFilter, resFilter, od, "cronjob", p.CronJob, p.CronJobs)
}

func ListDaemonSets(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	nsFilter NamespaceFilterOpts,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListNamespacedObj(p, connection, clusterIdentifier, nsFilter, resFilter, od, "daemonset", p.DaemonSet, p.DaemonSets)
}

// ListDeployments lits all deployments in the cluster.
func ListDeployments(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	nsFilter NamespaceFilterOpts,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListNamespacedObj(p, connection, clusterIdentifier, nsFilter, resFilter, od, "deployment", p.Deployment, p.Deployments)
}

// ListJobs list all jobs in the cluster.
func ListJobs(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	nsFilter NamespaceFilterOpts,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListNamespacedObj(p, connection, clusterIdentifier, nsFilter, resFilter, od, "job", p.Job, p.Jobs)
}

// ListPods list all pods in the cluster.
func ListPods(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	nsFilter NamespaceFilterOpts,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListNamespacedObj(p, connection, clusterIdentifier, nsFilter, resFilter, od, "pod", p.Pod, p.Pods)
}

// ListReplicaSets list all replicaSets in the cluster.
func ListReplicaSets(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	nsFilter NamespaceFilterOpts,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListNamespacedObj(p, connection, clusterIdentifier, nsFilter, resFilter, od, "replicaset", p.ReplicaSet, p.ReplicaSets)
}

// ListStatefulSets list all statefulsets in the cluster.
func ListStatefulSets(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	nsFilter NamespaceFilterOpts,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListNamespacedObj(p, connection, clusterIdentifier, nsFilter, resFilter, od, "statefulset", p.StatefulSet, p.StatefulSets)
}

func ListNamespacedObj[T runtime.Object](
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	nsFilter NamespaceFilterOpts,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
	workloadType string,
	getter func(string, string) (T, error),
	lister func(v1.Namespace) ([]T, error),
) ([]*asset.Asset, error) {
	workloads := []T{}

	if len(resFilter) > 0 {
		// If there is a resources filter we should only retrieve the workloads that are in the filter.
		if len(resFilter[workloadType]) == 0 {
			return []*asset.Asset{}, nil
		}

		for _, res := range resFilter[workloadType] {
			ds, err := getter(res.Namespace, res.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get %s %s/%s", workloadType, res.Namespace, res.Name)
			}

			workloads = append(workloads, ds)
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
					namespaces = append(namespaces, *n)
				}
			} else {
				return nil, errors.Wrap(err, "could not list kubernetes namespaces")
			}
		}

		for i := range namespaces {
			namespace := namespaces[i]
			skip, err := skipNamespace(namespace, nsFilter)
			if err != nil {
				log.Error().Err(err).Str("namespace", namespace.Name).Msg("error checking whether Namespace should be included or excluded")
				return nil, err
			}
			if skip {
				log.Debug().Str("namespace", namespace.Name).Msg("ignoring namespace")
				continue
			}

			workloadsPerNamespace, err := lister(namespace)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to list %ss", workloadType))
			}
			workloads = append(workloads, workloadsPerNamespace...)
		}
	}

	assetsIdx := map[string]*asset.Asset{}
	for i := range workloads {
		od.Add(workloads[i])

		asset, err := createAssetFromObject(workloads[i], p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to create asset from %s", workloadType))
		}

		// An error can never happen because of the type constraint.
		obj, _ := meta.Accessor(workloads[i])
		log.Debug().Str("name", obj.GetName()).Str("connection", asset.Connections[0].Host).Msgf("resolved %s", workloadType)

		assetsIdx[asset.PlatformIds[0]] = asset
	}

	// Return a unique list of assets. Manifests can contain a namespaces that is an empty string. When we try to list k8s
	// resources for the empty namespace, that actually means list all resources. Therefore we can have duplicate entries in the list.
	// Here we just return only the unique assets to make sure the code works correctly with both manifests and k8s API.
	assets := make([]*asset.Asset, 0, len(assetsIdx))
	for k := range assetsIdx {
		assets = append(assets, assetsIdx[k])
	}

	return assets, nil
}

func skipNamespace(namespace v1.Namespace, filter NamespaceFilterOpts) (bool, error) {
	// anything explicitly specified in the list of includes means accept only from that list
	if len(filter.include) > 0 {
		for _, ns := range filter.include {
			g, err := glob.Compile(ns)
			if err != nil {
				return false, err
			}
			if g.Match(namespace.Name) {
				// stop looking, we found our match
				return false, nil
			}
		}

		// didn't find it, so it must be skipped
		return true, nil
	}

	// if nothing explicitly meant to be included, then check whether
	// it should be excluded
	for _, ns := range filter.exclude {
		g, err := glob.Compile(ns)
		if err != nil {
			return false, err
		}
		if g.Match(namespace.Name) {
			return true, nil
		}
	}

	return false, nil
}
