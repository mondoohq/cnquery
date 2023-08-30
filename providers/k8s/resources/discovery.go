// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared/resources"
	admissionv1 "k8s.io/api/admission/v1"
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
		case DiscoveryClusters:
			assetId, err := conn.AssetId()
			if err != nil {
				return nil, err
			}
			list = []*inventory.Asset{
				{
					PlatformIds: []string{assetId},
					Name:        conn.Name(),
					Platform:    conn.Platform(),
					Connections: []*inventory.Config{invConfig}, // pass-in the parent connection config TODO: clone the config
				},
			}
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
		case DiscoveryAdmissionReviews:
			list, err = discoverAdmissionReviews(conn, invConfig, clusterId, k8s)
		case DiscoveryIngresses:
			list, err = discoverIngresses(invConfig, clusterId, k8s)
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
				shared.NewWorkloadPlatformId(clusterId, "replicaset", replicaset.Namespace.Data, replicaset.Name.Data, replicaset.Uid.Data),
			},
			Name: replicaset.Namespace.Data + "/" + replicaset.Name.Data,
			Platform: &inventory.Platform{
				Name:  "k8s-replicaset",
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
				shared.NewWorkloadPlatformId(clusterId, "daemonset", daemonset.Namespace.Data, daemonset.Name.Data, daemonset.Uid.Data),
			},
			Name: daemonset.Namespace.Data + "/" + daemonset.Name.Data,
			Platform: &inventory.Platform{
				Name:  "k8s-daemonset",
				Title: "Kubernetes DaemonSet, Kubernetes Cluster",
			},
			Labels:      labels,
			Connections: []*inventory.Config{invConfig}, // pass-in the parent connection config TODO: clone the config
		})
	}
	return assetList, nil
}

func discoverAdmissionReviews(conn shared.Connection, invConfig *inventory.Config, clusterId string, k8s *mqlK8s) ([]*inventory.Asset, error) {
	admissionReviews, err := conn.AdmissionReviews()
	if err != nil {
		return nil, err
	}

	var assetList []*inventory.Asset
	for i := range admissionReviews {
		aReview := admissionReviews[i]

		asset, err := assetFromAdmissionReview(aReview, conn.Platform().Runtime, invConfig, clusterId)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from admission review")
		}

		log.Debug().Str("connection", asset.Connections[0].Host).Msg("resolved AdmissionReview")

		assetList = append(assetList, asset)
	}

	return assetList, nil
}

func discoverIngresses(invConfig *inventory.Config, clusterId string, k8s *mqlK8s) ([]*inventory.Asset, error) {
	is := k8s.GetIngresses()
	if is.Error != nil {
		return nil, is.Error
	}

	assetList := make([]*inventory.Asset, 0, len(is.Data))
	for _, d := range is.Data {
		ingress := d.(*mqlK8sIngress)
		labels := map[string]string{}
		for k, v := range ingress.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &ingress.obj.ObjectMeta, clusterId)
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "ingress", ingress.Namespace.Data, ingress.Name.Data, ingress.Uid.Data),
			},
			Name: ingress.Namespace.Data + "/" + ingress.Name.Data,
			Platform: &inventory.Platform{
				Name:  "k8s-ingress",
				Title: "Kubernetes Ingress, Kubernetes Cluster",
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

func assetFromAdmissionReview(a admissionv1.AdmissionReview, runtime string, connection *inventory.Config, clusterIdentifier string) (*inventory.Asset, error) {
	// Use the meta from the request object.
	obj, err := resources.ResourcesFromManifest(bytes.NewReader(a.Request.Object.Raw))
	if err != nil {
		log.Error().Err(err).Msg("failed to parse object from admission review")
		return nil, err
	}
	objMeta, err := meta.Accessor(obj[0])
	if err != nil {
		log.Error().Err(err).Msg("could not access object attributes")
		return nil, err
	}
	objType, err := meta.TypeAccessor(&a)
	if err != nil {
		log.Error().Err(err).Msg("could not access object attributes")
		return nil, err
	}

	objectKind := objType.GetKind()
	platformData, err := createPlatformData(a.Kind, runtime)
	if err != nil {
		return nil, err
	}
	platformData.Version = objType.GetAPIVersion()
	platformData.Build = objMeta.GetResourceVersion()
	platformData.Labels = map[string]string{
		"uid": string(objMeta.GetUID()),
	}

	assetLabels := objMeta.GetLabels()
	if assetLabels == nil {
		assetLabels = map[string]string{}
	}
	ns := objMeta.GetNamespace()
	var name string
	if ns != "" {
		name = ns + "/" + objMeta.GetName()
		platformData.Labels["namespace"] = ns
	} else {
		name = objMeta.GetName()
	}

	addMondooAssetLabels(assetLabels, objMeta, clusterIdentifier)

	asset := &inventory.Asset{
		PlatformIds: []string{shared.NewWorkloadPlatformId(clusterIdentifier, strings.ToLower(objectKind), objMeta.GetNamespace(), objMeta.GetName(), string(objMeta.GetUID()))},
		Name:        name,
		Platform:    platformData,
		Connections: []*inventory.Config{connection},
		State:       inventory.State_STATE_ONLINE,
		Labels:      assetLabels,
	}

	return asset, nil
}

func createPlatformData(objectKind, runtime string) (*inventory.Platform, error) {
	platformData := &inventory.Platform{
		Family:  []string{"k8s"},
		Kind:    "k8s-object",
		Runtime: runtime,
	}

	switch objectKind {
	case "Node":
		platformData.Name = "k8s-node"
		platformData.Title = "Kubernetes Node"
	case "Pod":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-pod"
		platformData.Title = "Kubernetes Pod"
	case "CronJob":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-cronjob"
		platformData.Title = "Kubernetes CronJob"
	case "StatefulSet":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-statefulset"
		platformData.Title = "Kubernetes StatefulSet"
	case "Deployment":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-deployment"
		platformData.Title = "Kubernetes Deployment"
	case "Job":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-job"
		platformData.Title = "Kubernetes Job"
	case "ReplicaSet":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-replicaset"
		platformData.Title = "Kubernetes ReplicaSet"
	case "DaemonSet":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-daemonset"
		platformData.Title = "Kubernetes DaemonSet"
	case "AdmissionReview":
		platformData.Family = append(platformData.Family, "k8s-admission")
		platformData.Name = "k8s-admission"
		platformData.Title = "Kubernetes Admission Review"
	case "Ingress":
		platformData.Family = append(platformData.Family, "k8s-ingress")
		platformData.Name = "k8s-ingress"
		platformData.Title = "Kubernetes Ingress"
	case "Namespace":
		platformData.Family = append(platformData.Family, "k8s-namespace")
		platformData.Name = "k8s-namespace"
		platformData.Title = "Kubernetes Namespace"
	default:
		return nil, fmt.Errorf("could not determine object kind %s", objectKind)
	}
	return platformData, nil
}
