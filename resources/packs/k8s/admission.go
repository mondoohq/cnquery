package k8s

import (
	"bytes"
	"fmt"

	k8sResources "go.mondoo.com/cnquery/motor/providers/k8s/resources"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	v1 "k8s.io/api/admission/v1"
)

func (k *mqlK8sAdmissionreview) GetRequest() (interface{}, error) {
	kt, err := k8sProvider(k.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	result, err := kt.AdmissionReviews()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}

	// At the moment we don't support scanning >1 admission review at a time.
	if len(result) > 1 {
		return nil, fmt.Errorf("received more than 1 admission review")
	}

	aRequest := result[0].Request
	obj, err := k8sResources.ResourcesFromManifest(bytes.NewReader(aRequest.Object.Raw))
	if err != nil {
		return nil, err
	}

	objDict, err := core.JsonToDict(obj[0])
	if err != nil {
		return nil, err
	}

	args := []interface{}{
		"name", aRequest.Name,
		"namespace", aRequest.Namespace,
		"operation", string(aRequest.Operation),
		"object", objDict,
	}

	oldObj, err := k8sResources.ResourcesFromManifest(bytes.NewReader(aRequest.OldObject.Raw))
	if err != nil {
		return nil, err
	}

	if len(oldObj) == 1 {
		oldObjDict, err := core.JsonToDict(oldObj[0])
		if err != nil {
			return nil, err
		}
		args = append(args, "oldObject", oldObjDict)
	} else {
		args = append(args, "oldObject", nil)
	}

	r, err := k.MotorRuntime.CreateResource("k8s.admissionrequest", args...)
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
