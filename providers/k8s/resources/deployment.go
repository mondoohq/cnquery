package resources

import (
	"sync"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sDeploymentInternal struct {
	lock       sync.Mutex
	runtimeObj runtime.Object
	metaObj    metav1.Object
}

func (k *mqlK8s) deployments() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "deployments", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
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

		r, err := CreateResource(k.MqlRuntime, "k8s.deployment", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"namespace":       llx.StringData(obj.GetNamespace()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"manifest":        llx.DictData(manifest),
			"podSpec":         llx.DictData(podSpecDict),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sDeployment).runtimeObj = resource
		r.(*mqlK8sDeployment).metaObj = obj
		return r, nil
	})
}

func (k *mqlK8sDeployment) id() (string, error) {
	return k.Id.Data, nil
}

// func (p *mqlK8sDeployment) init(args *resources.Args) (*resources.Args, K8sDeployment, error) {
// 	return initNamespacedResource[K8sDeployment](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Deployments() })
// }

func (k *mqlK8sDeployment) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.metaObj.GetAnnotations()), nil
}

func (k *mqlK8sDeployment) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.metaObj.GetLabels()), nil
}

func (k *mqlK8sDeployment) initContainers() ([]interface{}, error) {
	return getContainers(k.runtimeObj, k.metaObj, k.MqlRuntime, InitContainerType)
}

func (k *mqlK8sDeployment) containers() ([]interface{}, error) {
	return getContainers(k.runtimeObj, k.metaObj, k.MqlRuntime, ContainerContainerType)
}
