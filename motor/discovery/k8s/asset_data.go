package k8s

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

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

	platformData := &platform.Platform{
		Family:  []string{"k8s", "k8s-workload"},
		Kind:    providers.Kind_KIND_K8S_OBJECT,
		Runtime: runtime,
	}
	// We need this at two places (discovery and tranport)
	// Here it is needed for the discovery and this is what ends up in  the database
	var objectKind string
	switch objectKind = objType.GetKind(); {
	case objectKind == "Pod":
		platformData.Name = "k8s-pod"
		platformData.Title = "Kubernetes Pod"
	case objectKind == "CronJob":
		platformData.Name = "k8s-cronjob"
		platformData.Title = "Kubernetes CronJob"
	case objectKind == "StatefulSet":
		platformData.Name = "k8s-statefulset"
		platformData.Title = "Kubernetes StatefulSet"
	case objectKind == "Deployment":
		platformData.Name = "k8s-deployment"
		platformData.Title = "Kubernetes Deployment"
	case objectKind == "Job":
		platformData.Name = "k8s-job"
		platformData.Title = "Kubernetes Job"
	case objectKind == "ReplicaSet":
		platformData.Name = "k8s-replicaset"
		platformData.Title = "Kubernetes ReplicaSet"
	case objectKind == "DaemonSet":
		platformData.Name = "k8s-daemonset"
		platformData.Title = "Kubernetes DaemonSet"
	default:
		return nil, errors.New("could not determine object kind")
	}
	platformData.Version = objType.GetAPIVersion()
	platformData.Build = objMeta.GetResourceVersion()
	platformData.Labels = map[string]string{
		"namespace": objMeta.GetNamespace(),
		"uid":       string(objMeta.GetUID()),
	}
	platformData.Kind = providers.Kind_KIND_K8S_OBJECT
	asset := &asset.Asset{
		PlatformIds: []string{k8s.NewPlatformWorkloadId(clusterIdentifier, strings.ToLower(objectKind), objMeta.GetNamespace(), objMeta.GetName())},
		Name:        objMeta.GetNamespace() + "/" + objMeta.GetName(),
		Platform:    platformData,
		Connections: []*providers.Config{connection},
		State:       asset.State_STATE_ONLINE,
		Labels:      objMeta.GetLabels(),
	}
	if asset.Labels == nil {
		asset.Labels = map[string]string{
			"namespace": objMeta.GetNamespace(),
		}
	} else {
		asset.Labels["namespace"] = objMeta.GetNamespace()
	}

	return asset, nil
}
