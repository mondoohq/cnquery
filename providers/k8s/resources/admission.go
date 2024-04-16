// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"fmt"
	"sync"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/shared/resources"
	admissionv1 "k8s.io/api/admission/v1"
)

type mqlK8sAdmissionrequestInternal struct {
	lock sync.Mutex
	obj  *admissionv1.AdmissionRequest
}

func (k *mqlK8sAdmissionreview) request() (*mqlK8sAdmissionrequest, error) {
	kt, err := k8sProvider(k.MqlRuntime.Connection)
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
	obj, err := resources.ResourcesFromManifest(bytes.NewReader(aRequest.Object.Raw))
	if err != nil {
		return nil, err
	}

	objDict, err := convert.JsonToDict(obj[0])
	if err != nil {
		return nil, err
	}

	args := map[string]*llx.RawData{
		"name":      llx.StringData(aRequest.Name),
		"namespace": llx.StringData(aRequest.Namespace),
		"operation": llx.StringData(string(aRequest.Operation)),
		"object":    llx.DictData(objDict),
	}

	oldObj, err := resources.ResourcesFromManifest(bytes.NewReader(aRequest.OldObject.Raw))
	if err != nil {
		return nil, err
	}

	if len(oldObj) == 1 {
		oldObjDict, err := convert.JsonToDict(oldObj[0])
		if err != nil {
			return nil, err
		}
		args["oldObject"] = llx.DictData(oldObjDict)
	} else {
		args["oldObject"] = llx.NilData
	}

	r, err := CreateResource(k.MqlRuntime, "k8s.admissionrequest", args)
	if err != nil {
		return nil, err
	}
	admReqRes := r.(*mqlK8sAdmissionrequest)
	admReqRes.obj = aRequest

	return admReqRes, nil
}

func (k *mqlK8sAdmissionrequest) userInfo() (*mqlK8sUserinfo, error) {
	userInfo := k.obj.UserInfo
	r, err := CreateResource(k.MqlRuntime, "k8s.userinfo", map[string]*llx.RawData{
		"username": llx.StringData(userInfo.Username),
		"uid":      llx.StringData(userInfo.UID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlK8sUserinfo), nil
}

func (k *mqlK8sAdmissionrequest) id() (string, error) {
	return k.Name.Data, nil
}

func (k *mqlK8sUserinfo) id() (string, error) {
	return k.Username.Data, nil
}
