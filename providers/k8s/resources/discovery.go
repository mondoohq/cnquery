package resources

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DiscoveryClusters         = "clusters"
	DiscoveryPods             = "pods"
	DiscoveryJobs             = "jobs"
	DiscoveryCronJobs         = "cronjobs"
	DiscoveryStatefulSets     = "statefulsets"
	DiscoveryDeployments      = "deployments"
	DiscoveryReplicaSets      = "replicasets"
	DiscoveryDaemonSets       = "daemonsets"
	DiscoveryContainerImages  = "container-images"
	DiscoveryAdmissionReviews = "admissionreviews"
	DiscoveryIngresses        = "ingresses"
	DiscoveryNamespaces       = "namespaces"
)

func Discover(runtime *plugin.Runtime) (*inventory.Inventory, error) {
	conn := runtime.Connection.(shared.Connection)

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	invConfig := conn.InventoryConfig()

	res, err := runtime.CreateResource(runtime, "k8s", nil)
	if err != nil {
		return nil, err
	}
	k8s := res.(*mqlK8s)

	clusterId, err := conn.AssetId()
	if err != nil {
		return nil, err
	}
	for _, target := range invConfig.Discover.Targets {
		var list []*inventory.Asset
		switch target {
		case DiscoveryPods:
			list, err = discoverPods(invConfig, clusterId, k8s)
		case DiscoveryJobs:
			list, err = discoverJobs(invConfig, clusterId, k8s)
		case DiscoveryCronJobs:
			list, err = discoverCronJobs(invConfig, clusterId, k8s)
		case DiscoveryStatefulSets:
			list, err = discoverStatefulSets(invConfig, clusterId, k8s)
		case DiscoveryDeployments:
			list, err = discoverDeployments(invConfig, clusterId, k8s)
		case DiscoveryReplicaSets:
			list, err = discoverReplicaSets(invConfig, clusterId, k8s)
		case DiscoveryDaemonSets:
			list, err = discoverDaemonSets(invConfig, clusterId, k8s)
		default:
			continue
		}

		if err != nil {
			return nil, err
		}
		in.Spec.Assets = append(in.Spec.Assets, list...)
	}
	return in, nil
}

func discoverPods(invConfig *inventory.Config, clusterId string, k8s *mqlK8s) ([]*inventory.Asset, error) {
	pods := k8s.GetPods()
	if pods.Error != nil {
		return nil, pods.Error
	}

	assetList := make([]*inventory.Asset, 0, len(pods.Data))
	for _, p := range pods.Data {
		pod := p.(*mqlK8sPod)
		labels := map[string]string{}
		for k, v := range pod.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &pod.obj.ObjectMeta, clusterId)
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "pod", pod.Namespace.Data, pod.Name.Data, pod.Uid.Data),
			},
			Name: pod.Namespace.Data + "/" + pod.Name.Data,
			Platform: &inventory.Platform{
				Name:  "k8s-pod",
				Title: "Kubernetes Pod, Kubernetes Cluster",
			},
			Labels:      labels,
			Connections: []*inventory.Config{invConfig}, // pass-in the parent connection config TODO: clone the config
		})
	}
	return assetList, nil
}

func discoverJobs(invConfig *inventory.Config, clusterId string, k8s *mqlK8s) ([]*inventory.Asset, error) {
	jobs := k8s.GetJobs()
	if jobs.Error != nil {
		return nil, jobs.Error
	}

	assetList := make([]*inventory.Asset, 0, len(jobs.Data))
	for _, j := range jobs.Data {
		job := j.(*mqlK8sJob)
		labels := map[string]string{}
		for k, v := range job.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &job.obj.ObjectMeta, clusterId)
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "job", job.Namespace.Data, job.Name.Data, job.Uid.Data),
			},
			Name: job.Namespace.Data + "/" + job.Name.Data,
			Platform: &inventory.Platform{
				Name:  "k8s-job",
				Title: "Kubernetes Job, Kubernetes Cluster",
			},
			Labels:      labels,
			Connections: []*inventory.Config{invConfig}, // pass-in the parent connection config TODO: clone the config
		})
	}
	return assetList, nil
}

func discoverCronJobs(invConfig *inventory.Config, clusterId string, k8s *mqlK8s) ([]*inventory.Asset, error) {
	cjs := k8s.GetCronjobs()
	if cjs.Error != nil {
		return nil, cjs.Error
	}

	assetList := make([]*inventory.Asset, 0, len(cjs.Data))
	for _, cj := range cjs.Data {
		cjob := cj.(*mqlK8sCronjob)
		labels := map[string]string{}
		for k, v := range cjob.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &cjob.obj.ObjectMeta, clusterId)
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "cronjob", cjob.Namespace.Data, cjob.Name.Data, cjob.Uid.Data),
			},
			Name: cjob.Namespace.Data + "/" + cjob.Name.Data,
			Platform: &inventory.Platform{
				Name:  "k8s-cronjob",
				Title: "Kubernetes CronJob, Kubernetes Cluster",
			},
			Labels:      labels,
			Connections: []*inventory.Config{invConfig}, // pass-in the parent connection config TODO: clone the config
		})
	}
	return assetList, nil
}

func discoverStatefulSets(invConfig *inventory.Config, clusterId string, k8s *mqlK8s) ([]*inventory.Asset, error) {
	ss := k8s.GetStatefulsets()
	if ss.Error != nil {
		return nil, ss.Error
	}

	assetList := make([]*inventory.Asset, 0, len(ss.Data))
	for _, j := range ss.Data {
		statefulset := j.(*mqlK8sStatefulset)
		labels := map[string]string{}
		for k, v := range statefulset.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &statefulset.obj.ObjectMeta, clusterId)
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "statefulset", statefulset.Namespace.Data, statefulset.Name.Data, statefulset.Uid.Data),
			},
			Name: statefulset.Namespace.Data + "/" + statefulset.Name.Data,
			Platform: &inventory.Platform{
				Name:  "k8s-statefulset",
				Title: "Kubernetes StatefulSet, Kubernetes Cluster",
			},
			Labels:      labels,
			Connections: []*inventory.Config{invConfig}, // pass-in the parent connection config TODO: clone the config
		})
	}
	return assetList, nil
}

func discoverDeployments(invConfig *inventory.Config, clusterId string, k8s *mqlK8s) ([]*inventory.Asset, error) {
	ds := k8s.GetStatefulsets()
	if ds.Error != nil {
		return nil, ds.Error
	}

	assetList := make([]*inventory.Asset, 0, len(ds.Data))
	for _, d := range ds.Data {
		deployment := d.(*mqlK8sStatefulset)
		labels := map[string]string{}
		for k, v := range deployment.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &deployment.obj.ObjectMeta, clusterId)
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "statefulset", deployment.Namespace.Data, deployment.Name.Data, deployment.Uid.Data),
			},
			Name: deployment.Namespace.Data + "/" + deployment.Name.Data,
			Platform: &inventory.Platform{
				Name:  "k8s-deployment",
				Title: "Kubernetes Deployment, Kubernetes Cluster",
			},
			Labels:      labels,
			Connections: []*inventory.Config{invConfig}, // pass-in the parent connection config TODO: clone the config
		})
	}
	return assetList, nil
}

func discoverReplicaSets(invConfig *inventory.Config, clusterId string, k8s *mqlK8s) ([]*inventory.Asset, error) {
	rs := k8s.GetReplicasets()
	if rs.Error != nil {
		return nil, rs.Error
	}

	assetList := make([]*inventory.Asset, 0, len(rs.Data))
	for _, r := range rs.Data {
		replicaset := r.(*mqlK8sReplicaset)
		labels := map[string]string{}
		for k, v := range replicaset.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &replicaset.obj.ObjectMeta, clusterId)
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "statefulset", replicaset.Namespace.Data, replicaset.Name.Data, replicaset.Uid.Data),
			},
			Name: replicaset.Namespace.Data + "/" + replicaset.Name.Data,
			Platform: &inventory.Platform{
				Name:  "k8s-statefulset",
				Title: "Kubernetes ReplicaSet, Kubernetes Cluster",
			},
			Labels:      labels,
			Connections: []*inventory.Config{invConfig}, // pass-in the parent connection config TODO: clone the config
		})
	}
	return assetList, nil
}

func discoverDaemonSets(invConfig *inventory.Config, clusterId string, k8s *mqlK8s) ([]*inventory.Asset, error) {
	ds := k8s.GetDaemonsets()
	if ds.Error != nil {
		return nil, ds.Error
	}

	assetList := make([]*inventory.Asset, 0, len(ds.Data))
	for _, d := range ds.Data {
		daemonset := d.(*mqlK8sDaemonset)
		labels := map[string]string{}
		for k, v := range daemonset.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &daemonset.obj.ObjectMeta, clusterId)
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "statefulset", daemonset.Namespace.Data, daemonset.Name.Data, daemonset.Uid.Data),
			},
			Name: daemonset.Namespace.Data + "/" + daemonset.Name.Data,
			Platform: &inventory.Platform{
				Name:  "k8s-statefulset",
				Title: "Kubernetes DaemonSet, Kubernetes Cluster",
			},
			Labels:      labels,
			Connections: []*inventory.Config{invConfig}, // pass-in the parent connection config TODO: clone the config
		})
	}
	return assetList, nil
}

func addMondooAssetLabels(assetLabels map[string]string, objMeta metav1.Object, clusterIdentifier string) {
	ns := objMeta.GetNamespace()
	if ns != "" {
		assetLabels["k8s.mondoo.com/namespace"] = ns
	}
	assetLabels["k8s.mondoo.com/name"] = objMeta.GetName()
	if string(objMeta.GetUID()) != "" {
		// objects discovered from manifest do not necessarily have a UID
		assetLabels["k8s.mondoo.com/uid"] = string(objMeta.GetUID())
	}
	objType, err := meta.TypeAccessor(objMeta)
	if err == nil {
		assetLabels["k8s.mondoo.com/kind"] = objType.GetKind()
		assetLabels["k8s.mondoo.com/apiVersion"] = objType.GetAPIVersion()
	}
	if objMeta.GetResourceVersion() != "" {
		// objects discovered from manifest do not necessarily have a resource version
		assetLabels["k8s.mondoo.com/resource-version"] = objMeta.GetResourceVersion()
	}
	assetLabels["k8s.mondoo.com/cluster-id"] = clusterIdentifier

	owners := objMeta.GetOwnerReferences()
	if len(owners) > 0 {
		owner := owners[0]
		assetLabels["k8s.mondoo.com/owner-kind"] = owner.Kind
		assetLabels["k8s.mondoo.com/owner-name"] = owner.Name
		assetLabels["k8s.mondoo.com/owner-uid"] = string(owner.UID)
	}
}
