package k8s

import (
	"errors"

	k8s_resources "go.mondoo.com/cnquery/motor/providers/k8s/resources"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetPods() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "pods.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
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

		r, err := k.MotorRuntime.CreateResource("k8s.pod",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"labels", core.StrMapToInterface(obj.GetLabels()),
			"annotations", core.StrMapToInterface(obj.GetAnnotations()),
			"apiVersion", objT.GetAPIVersion(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"podSpec", podSpecDict,
			"manifest", manifest,
		)
		if err != nil {
			return nil, err
		}
		r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *mqlK8sPod) id() (string, error) {
	return k.Id()
}

func (p *mqlK8sPod) init(args *resources.Args) (*resources.Args, K8sPod, error) {
	return initNamespacedResource[K8sPod](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Pods() })
}

func (k *mqlK8sPod) GetInitContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, InitContainerType)
}

func (k *mqlK8sPod) GetContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, ContainerContainerType)
}

func (k *mqlK8sPod) GetNamespace() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (k *mqlK8sPod) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sPod) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}

func (k *mqlK8sPod) GetNode() (K8sNode, error) {
	rawSpec, err := k.PodSpec()
	if err != nil {
		return nil, err
	}

	podSpec, ok := rawSpec.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid pod spec information")
	}

	obj, err := k.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, err
	}
	k8sResource := obj.(K8s)

	nodes, err := k8sResource.Nodes()
	if err != nil {
		return nil, err
	}

	matchFn := func(node K8sNode) bool {
		name, _ := node.Name()
		if name == podSpec["nodeName"] {
			return true
		}
		return false
	}

	for i := range nodes {
		node := nodes[i].(K8sNode)
		if matchFn(node) {
			return node, nil
		}
	}

	return nil, nil
}
