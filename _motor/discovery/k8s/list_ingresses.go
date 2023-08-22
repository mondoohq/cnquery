// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package k8s

import (
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
	return ListNamespacedObj(p, connection, clusterIdentifier, nsFilter, resFilter, od, "ingress", p.Ingress, p.Ingresses)
}
