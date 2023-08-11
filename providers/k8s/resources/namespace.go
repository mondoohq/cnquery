package resources

import (
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) namespaces() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "namespaces", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := convert.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		return CreateResource(k.MqlRuntime, "k8s.namespace", map[string]*llx.RawData{
			"id":       llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":      llx.StringData(string(obj.GetUID())),
			"name":     llx.StringData(obj.GetName()),
			"created":  llx.TimeData(ts.Time),
			"manifest": llx.DictData(manifest),
		})
	})
}

func (k *mqlK8sNamespace) id() (string, error) {
	return k.Id.Data, nil
}
