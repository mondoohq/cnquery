package k8s

import (
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetNamespaces() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "namespaces", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		return k.MotorRuntime.CreateResource("k8s.namespace",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"apiVersion", objT.GetAPIVersion(),
			"kind", objT.GetKind(),
			"name", obj.GetName(),
			"labels", core.StrMapToInterface(obj.GetLabels()),
			"annotations", core.StrMapToInterface(obj.GetAnnotations()),
			"created", &ts.Time,
			"manifest", manifest,
		)
	})
}

func (p *mqlK8sNamespace) init(args *resources.Args) (*resources.Args, K8sNamespace, error) {
	return initResource[K8sNamespace](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Namespaces() })
}

func (k *mqlK8sNamespace) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sNamespace) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}

func (k *mqlK8sNamespace) id() (string, error) {
	return k.Id()
}
