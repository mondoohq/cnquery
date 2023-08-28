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
		switch target {
		case DiscoveryPods:
		}
		list, err := discoverPods(invConfig, clusterId, k8s)
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
		for k, v := range pod.Labels.Data {
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
