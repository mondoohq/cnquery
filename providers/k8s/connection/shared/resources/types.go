package resources

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func UnstructuredListToObjectList(list []unstructured.Unstructured) []runtime.Object {
	var objs []runtime.Object
	for i := range list {
		objs = append(objs, ConvertToK8sObject(list[i]))
	}
	return objs
}

// ConvertToK8sObject converts an unstructured object to a Kubernetes runtime object if the schema
// recognizes it. If the object is not registered in the schema it will remain unstructured.
func ConvertToK8sObject(r unstructured.Unstructured) runtime.Object {
	// Get a struct with the correct type of object
	obj, err := ClientSchema().New(r.GroupVersionKind())
	if err != nil {
		return &r
	}

	// Convert the unstructured struct to the real object
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(r.Object, obj); err != nil {
		return &r
	}
	return obj
}
