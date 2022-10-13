package k8s

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// ListCronJobs list all cronjobs in the cluster.
func ListCronJobs(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListWorkloads(p, connection, clusterIdentifier, namespaceFilter, resFilter, od, "cronjob", p.CronJob, p.CronJobs)
}

func ListDaemonSets(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListWorkloads(p, connection, clusterIdentifier, namespaceFilter, resFilter, od, "daemonset", p.DaemonSet, p.DaemonSets)
}

// ListDeployments lits all deployments in the cluster.
func ListDeployments(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListWorkloads(p, connection, clusterIdentifier, namespaceFilter, resFilter, od, "deployment", p.Deployment, p.Deployments)
}

// ListJobs list all jobs in the cluster.
func ListJobs(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListWorkloads(p, connection, clusterIdentifier, namespaceFilter, resFilter, od, "job", p.Job, p.Jobs)
}

// ListPods list all pods in the cluster.
func ListPods(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListWorkloads(p, connection, clusterIdentifier, namespaceFilter, resFilter, od, "pod", p.Pod, p.Pods)
}

// ListReplicaSets list all replicaSets in the cluster.
func ListReplicaSets(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListWorkloads(p, connection, clusterIdentifier, namespaceFilter, resFilter, od, "replicaset", p.ReplicaSet, p.ReplicaSets)
}

// ListStatefulSets list all statefulsets in the cluster.
func ListStatefulSets(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	return ListWorkloads(p, connection, clusterIdentifier, namespaceFilter, resFilter, od, "statefulset", p.StatefulSet, p.StatefulSets)
}

func ListWorkloads[T runtime.Object](
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter string,
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
			return nil, errors.Wrap(err, "could not list kubernetes namespaces")
		}

		for i := range namespaces {
			namespace := namespaces[i]
			if namespaceFilter != "" && namespace.Name != namespaceFilter {
				log.Debug().Str("namespace", namespace.Name).Str("filter", namespaceFilter).Msg("namespace not included")
				continue
			}

			workloadsPerNamespace, err := lister(namespace)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to list %ss", workloadType))
			}
			workloads = append(workloads, workloadsPerNamespace...)
		}
	}

	assets := []*asset.Asset{}
	for i := range workloads {
		od.Add(workloads[i])

		asset, err := createAssetFromObject(workloads[i], p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to create asset from %s", workloadType))
		}

		// An error can never happen because of the type constraint.
		obj, _ := meta.Accessor(workloads[i])
		log.Debug().Str("name", obj.GetName()).Str("connection", asset.Connections[0].Host).Msgf("resolved %s", workloadType)

		assets = append(assets, asset)
	}

	return assets, nil
}
