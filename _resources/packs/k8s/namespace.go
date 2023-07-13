package k8s

import (
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
			"name", obj.GetName(),
			"created", &ts.Time,
			"manifest", manifest,
		)
	})
}

func (k *mqlK8sNamespace) id() (string, error) {
	return k.Id()
}
