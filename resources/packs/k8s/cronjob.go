package k8s

import (
	"errors"

	k8s_resources "go.mondoo.io/mondoo/motor/providers/k8s/resources"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetCronjobs() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "cronjobs", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		podSpec, err := k8s_resources.GetPodSpec(resource)
		if err != nil {
			return nil, err
		}

		podSpecDict, err := core.JsonToDict(podSpec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.cronjob",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"podSpec", podSpecDict,
		)
		if err != nil {
			return nil, err
		}
		r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *mqlK8sCronjob) id() (string, error) {
	return k.Id()
}

func (p *mqlK8sCronjob) init(args *resources.Args) (*resources.Args, K8sCronjob, error) {
	return initNamespacedResource[K8sCronjob](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Cronjobs() })
}

func (k *mqlK8sCronjob) GetNamespace() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (k *mqlK8sCronjob) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sCronjob) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}

func (k *mqlK8sCronjob) GetInitContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, InitContainerType)
}

func (k *mqlK8sCronjob) GetContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, ContainerContainerType)
}
