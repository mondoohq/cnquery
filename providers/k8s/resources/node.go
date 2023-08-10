package resources

import (
	"sync"

	"go.mondoo.com/cnquery/llx"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sNodeInternal struct {
	lock    sync.Mutex
	metaObj metav1.Object
}

func (k *mqlK8s) nodes() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "nodes.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		r, err := CreateResource(k.MqlRuntime, "k8s.node", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"kind":            llx.StringData(objT.GetKind()),
		})
		if err != nil {
			return nil, err
		}

		r.(*mqlK8sNode).metaObj = obj

		return r, nil
	})
}

func (k *mqlK8sNode) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sNode) annotations() (map[string]interface{}, error) {
	return MapToInterfaceMap(k.metaObj.GetAnnotations()), nil
}

func (k *mqlK8sNode) labels() (map[string]interface{}, error) {
	return MapToInterfaceMap(k.metaObj.GetLabels()), nil
}
