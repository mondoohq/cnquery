// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

// SerializableInternal implementations for k8s resources.
// These allow the SQLite-backed resource cache to persist and restore
// internal state (k8s API objects) that is set imperatively after
// resource creation (e.g., r.(*mqlK8sNamespace).obj = &ns).

import (
	"encoding/json"

	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ---------------------------------------------------------------------------
// Concrete typed pointers — direct JSON marshal/unmarshal
// ---------------------------------------------------------------------------

// Node

func (k *mqlK8sNode) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sNode) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj corev1.Node
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// Namespace

func (k *mqlK8sNamespace) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sNamespace) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj corev1.Namespace
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// ServiceAccount

func (k *mqlK8sServiceaccount) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sServiceaccount) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj corev1.ServiceAccount
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// Role

func (k *mqlK8sRbacRole) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sRbacRole) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj rbacv1.Role
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// ClusterRole

func (k *mqlK8sRbacClusterrole) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sRbacClusterrole) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj rbacv1.ClusterRole
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// RoleBinding

func (k *mqlK8sRbacRolebinding) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sRbacRolebinding) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj rbacv1.RoleBinding
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// ClusterRoleBinding

func (k *mqlK8sRbacClusterrolebinding) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sRbacClusterrolebinding) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj rbacv1.ClusterRoleBinding
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// NetworkPolicy

func (k *mqlK8sNetworkpolicy) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sNetworkpolicy) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj networkingv1.NetworkPolicy
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// AdmissionRequest

func (k *mqlK8sAdmissionrequest) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sAdmissionrequest) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj admissionv1.AdmissionRequest
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// ValidatingWebhookConfiguration

func (k *mqlK8sAdmissionValidatingwebhookconfiguration) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sAdmissionValidatingwebhookconfiguration) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj admissionregistrationv1.ValidatingWebhookConfiguration
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// ---------------------------------------------------------------------------
// Secret — concrete type + metaObj recovery
// Secret.metaObj is the same as Secret.obj (Secret implements metav1.Object)
// ---------------------------------------------------------------------------

func (k *mqlK8sSecret) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sSecret) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj corev1.Secret
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	k.metaObj = &obj // Secret embeds ObjectMeta, implements metav1.Object
	return nil
}

// ---------------------------------------------------------------------------
// runtime.Object types — unmarshal into known concrete type
// ---------------------------------------------------------------------------

// Pod

func (k *mqlK8sPod) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sPod) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj corev1.Pod
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// Service

func (k *mqlK8sService) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sService) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj corev1.Service
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// ConfigMap

func (k *mqlK8sConfigmap) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sConfigmap) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj corev1.ConfigMap
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// Deployment

func (k *mqlK8sDeployment) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sDeployment) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj appsv1.Deployment
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// DaemonSet

func (k *mqlK8sDaemonset) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sDaemonset) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj appsv1.DaemonSet
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// StatefulSet

func (k *mqlK8sStatefulset) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sStatefulset) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj appsv1.StatefulSet
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// ReplicaSet

func (k *mqlK8sReplicaset) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sReplicaset) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj appsv1.ReplicaSet
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// Job

func (k *mqlK8sJob) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sJob) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj batchv1.Job
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// CronJob

func (k *mqlK8sCronjob) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sCronjob) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj batchv1.CronJob
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}

// ---------------------------------------------------------------------------
// Ingress — runtime.Object + extra objId field
// ---------------------------------------------------------------------------

type k8sIngressInternalData struct {
	Obj   json.RawMessage `json:"o"`
	ObjID string          `json:"i"`
}

func (k *mqlK8sIngress) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	objJSON, err := json.Marshal(k.obj)
	if err != nil {
		return nil, err
	}
	return json.Marshal(k8sIngressInternalData{
		Obj:   objJSON,
		ObjID: k.objId,
	})
}

func (k *mqlK8sIngress) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var w k8sIngressInternalData
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	var obj networkingv1.Ingress
	if err := json.Unmarshal(w.Obj, &obj); err != nil {
		return err
	}
	k.obj = &obj
	k.objId = w.ObjID
	return nil
}

// ---------------------------------------------------------------------------
// CustomResource — metav1.Object (concrete type: *unstructured.Unstructured)
// ---------------------------------------------------------------------------

func (k *mqlK8sCustomresource) MarshalInternal() ([]byte, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.obj == nil {
		return nil, nil
	}
	return json.Marshal(k.obj)
}

func (k *mqlK8sCustomresource) UnmarshalInternal(data []byte) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	var obj unstructured.Unstructured
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	k.obj = &obj
	return nil
}
