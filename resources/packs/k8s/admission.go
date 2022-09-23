package k8s

import (
	"bytes"
	"fmt"

	k8sResources "go.mondoo.com/cnquery/motor/providers/k8s/resources"
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

func (k *mqlK8sAdmissionreview) GetRequest() (interface{}, error) {
	entry, ok := k.Cache.Load("_resource")
	if !ok {
		return nil, fmt.Errorf("failed to load AdmissionReview resource from cache")
	}

	a, ok := entry.Data.(v1.AdmissionReview)
	if !ok {
		return nil, fmt.Errorf("failed to convert cache entrry to AdmissionReview")
	}

	aRequest := a.Request
	obj, err := k8sResources.ResourcesFromManifest(bytes.NewReader(aRequest.Object.Raw))
	if err != nil {
		return nil, err
	}

	objDict, err := core.JsonToDictSlice(obj)
	if err != nil {
		return nil, err
	}

	oldObj, err := k8sResources.ResourcesFromManifest(bytes.NewReader(aRequest.OldObject.Raw))
	if err != nil {
		return nil, err
	}

	oldObjDict, err := core.JsonToDictSlice(oldObj)
	if err != nil {
		return nil, err
	}

	r, err := k.MotorRuntime.CreateResource("k8s.admissionrequest",
		"name", aRequest.Name,
		"namespace", aRequest.Namespace,
		"operation", string(aRequest.Operation),
		"object", objDict,
		"oldObject", oldObjDict)
	if err != nil {
		return nil, err
	}
	r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: aRequest})

	return r, nil
}

func (k *mqlK8sAdmissionrequest) GetUserInfo() (interface{}, error) {
	entry, ok := k.Cache.Load("_resource")
	if !ok {
		return nil, fmt.Errorf("failed to load AdmissionRequest resource from cache")
	}

	a, ok := entry.Data.(*v1.AdmissionRequest)
	if !ok {
		return nil, fmt.Errorf("failed to convert cache entrry to AdmissionRequest")
	}

	userInfo := a.UserInfo
	return k.MotorRuntime.CreateResource("k8s.userinfo",
		"username", userInfo.Username,
		"uid", userInfo.UID)
}

func (k *mqlK8sAdmissionreview) id() (string, error) {
	return "admissionreview", nil
}

func (k *mqlK8sAdmissionrequest) id() (string, error) {
	return k.Name()
}

func (k *mqlK8sUserinfo) id() (string, error) {
	return k.Name, nil
}
