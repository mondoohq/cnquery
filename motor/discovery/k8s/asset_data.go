package k8s

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func createPlatformData(objectKind, runtime string) (*platform.Platform, error) {
	platformData := &platform.Platform{
		Family:  []string{"k8s"},
		Kind:    providers.Kind_KIND_K8S_OBJECT,
		Runtime: runtime,
	}
	// We need this at two places (discovery and tranport)
	// Here it is needed for the discovery and this is what ends up in  the database
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
	default:
		return nil, fmt.Errorf("could not determine object kind %s", objectKind)
	}
	return platformData, nil
}

func createAssetFromObject(object runtime.Object, runtime string, connection *providers.Config, clusterIdentifier string) (*asset.Asset, error) {
	objMeta, err := meta.Accessor(object)
	if err != nil {
		log.Error().Err(err).Msg("could not access object attributes")
		return nil, err
	}
	objType, err := meta.TypeAccessor(object)
	if err != nil {
		log.Error().Err(err).Msg("could not access object attributes")
		return nil, err
	}

	objectKind := objType.GetKind()
	platformData, err := createPlatformData(objectKind, runtime)
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

	addMondooAssetLabels(assetLabels, objMeta, objType, clusterIdentifier)

	newConnection := proto.Clone(connection).(*providers.Config)
	newConnection.Options = map[string]string{}
	for k, v := range connection.Options {
		newConnection.Options[k] = v
	}
	newConnection.Options["object-kind"] = strings.ToLower(objectKind)

	asset := &asset.Asset{
		PlatformIds: []string{k8s.NewPlatformWorkloadId(clusterIdentifier, strings.ToLower(objectKind), objMeta.GetNamespace(), objMeta.GetName())},
		Name:        name,
		Platform:    platformData,
		Connections: []*providers.Config{newConnection},
		State:       asset.State_STATE_ONLINE,
		Labels:      assetLabels,
	}

	return asset, nil
}

func addMondooAssetLabels(assetLabels map[string]string, objMeta v1.Object, objType meta.Type, clusterIdentifier string) {
	ns := objMeta.GetNamespace()
	if ns != "" {
		assetLabels["k8s.mondoo.com/namespace"] = ns
	}
	assetLabels["k8s.mondoo.com/name"] = objMeta.GetName()
	if string(objMeta.GetUID()) != "" {
		// objects discovered from manifest do not necessarily have a UID
		assetLabels["k8s.mondoo.com/uid"] = string(objMeta.GetUID())
	}
	assetLabels["k8s.mondoo.com/kind"] = objType.GetKind()
	assetLabels["k8s.mondoo.com/apiVersion"] = objType.GetAPIVersion()
	if objMeta.GetResourceVersion() != "" {
		// objects discovered from manifest do not necessarily have a resource version
		assetLabels["k8s.mondoo.com/resourceVersion"] = objMeta.GetResourceVersion()
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
