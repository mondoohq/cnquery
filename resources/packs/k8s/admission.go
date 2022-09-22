package k8s

import (
	"fmt"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	v1 "k8s.io/api/admission/v1"
)

func (k *mqlK8s) GetAdmissionreviews() ([]interface{}, error) {
	kt, err := k8sProvider(k.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	result, err := kt.AdmissionReviews()
	if err != nil {
		return nil, err
	}

	resp := make([]interface{}, 0, len(result))
	for _, a := range result {
		r, err := k.MotorRuntime.CreateResource("k8s.admissionreview")
		if err != nil {
			return nil, err
		}
		r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: a})

		resp = append(resp, r)
	}

	return resp, nil
}

func (k *mqlK8sAdmissionReview) GetRequest() (interface{}, error) {
	entry, ok := k.Cache.Load("_resource")
	if !ok {
		return nil, fmt.Errorf("failed to load AdmissionReview resource from cache")
	}

	a, ok := entry.Data.(v1.AdmissionReview)
	if !ok {
		return nil, fmt.Errorf("failed to convert cache entrry to AdmissionReview")
	}

	aRequest := a.Request

	obj, err := core.JsonToDictSlice(aRequest.Object)
	if err != nil {
		return nil, err
	}

	oldObj, err := core.JsonToDictSlice(aRequest.OldObject)
	if err != nil {
		return nil, err
	}

	return k.MotorRuntime.CreateResource("k8s.admissionreview",
		"name", aRequest.Name,
		"namespace", aRequest.Namespace,
		"operation", aRequest.Operation,
		"object", obj,
		"oldObject", oldObj)
}

func (k *mqlK8sAdmissionRequest) GetUserInfo() (interface{}, error) {
	return nil, nil
}

func (k *mqlK8sAdmissionReview) id() (string, error) {
	return k.Name, nil
}

func (k *mqlK8sAdmissionRequest) id() (string, error) {
	return k.Name()
}

func (k *mqlK8sUserInfo) id() (string, error) {
	return k.Name, nil
}
