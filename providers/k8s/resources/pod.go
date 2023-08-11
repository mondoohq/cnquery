package resources

import (
	"sync"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sPodInternal struct {
	lock       sync.Mutex
	runtimeObj runtime.Object
	metaObj    metav1.Object
}

func (k *mqlK8s) pods() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "pods.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := convert.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		podSpec, err := resources.GetPodSpec(resource)
		if err != nil {
			return nil, err
		}

		podSpecDict, err := convert.JsonToDict(podSpec)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.pod", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"namespace":       llx.StringData(obj.GetNamespace()),
			"apiVersion":      llx.StringData(objT.GetAPIVersion()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"podSpec":         llx.DictData(podSpecDict),
			"manifest":        llx.DictData(manifest),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sPod).runtimeObj = resource
		r.(*mqlK8sPod).metaObj = obj
		return r, nil
	})
}

func (k *mqlK8sPod) id() (string, error) {
	return k.Id.Data, nil
}

// func (p *mqlK8sPod) init(args *resources.Args) (*resources.Args, K8sPod, error) {
// 	return initNamespacedResource[K8sPod](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Pods() })
// }

func (k *mqlK8sPod) initContainers() ([]interface{}, error) {
	return getContainers(k.runtimeObj, k.metaObj, k.MqlRuntime, InitContainerType)
}

func (k *mqlK8sPod) ephemeralContainers() ([]interface{}, error) {
	return getContainers(k.runtimeObj, k.metaObj, k.MqlRuntime, EphemeralContainerType)
}

func (k *mqlK8sPod) containers() ([]interface{}, error) {
	return getContainers(k.runtimeObj, k.metaObj, k.MqlRuntime, ContainerContainerType)
}

func (k *mqlK8sPod) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.metaObj.GetAnnotations()), nil
}

func (k *mqlK8sPod) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.metaObj.GetLabels()), nil
}

func (k *mqlK8sPod) node() (*mqlK8sNode, error) {
	podSpec, err := resources.GetPodSpec(k.runtimeObj)
	if err != nil {
		return nil, err
	}

	node, err := NewResource(k.MqlRuntime, "k8s.node", map[string]*llx.RawData{
		"name": llx.StringData(podSpec.NodeName),
	})
	if err != nil {
		return nil, err
	}

	return node.(*mqlK8sNode), nil
}
