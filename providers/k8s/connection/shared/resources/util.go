// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func GetPodSpec(obj runtime.Object) (*corev1.PodSpec, error) {
	var podSpec *corev1.PodSpec
	switch x := obj.(type) {
	case *batchv1.Job:
		podSpec = &x.Spec.Template.Spec
	case *batchv1.CronJob:
		podSpec = &x.Spec.JobTemplate.Spec.Template.Spec
	case *batchv1beta1.CronJob:
		podSpec = &x.Spec.JobTemplate.Spec.Template.Spec
	case *appsv1.DaemonSet:
		podSpec = &x.Spec.Template.Spec
	case *extensionsv1beta1.DaemonSet:
		podSpec = &x.Spec.Template.Spec
	case *appsv1beta2.DaemonSet:
		podSpec = &x.Spec.Template.Spec
	case *appsv1.Deployment:
		podSpec = &x.Spec.Template.Spec
	case *appsv1beta1.Deployment:
		podSpec = &x.Spec.Template.Spec
	case *appsv1beta2.Deployment:
		podSpec = &x.Spec.Template.Spec
	case *corev1.PodTemplate:
		podSpec = &x.Template.Spec
	case *corev1.Pod:
		podSpec = &x.Spec
	case *corev1.ReplicationController:
		podSpec = &x.Spec.Template.Spec
	case *appsv1.StatefulSet:
		podSpec = &x.Spec.Template.Spec
	case *appsv1beta1.StatefulSet:
		podSpec = &x.Spec.Template.Spec
	case *appsv1.ReplicaSet:
		podSpec = &x.Spec.Template.Spec
	case *unstructured.Unstructured:
		gvk := x.GetObjectKind().GroupVersionKind()
		return nil, fmt.Errorf("object %s with version %s/%s is not supported", gvk.Kind, gvk.Group, gvk.Version)
	default:
		return nil, fmt.Errorf("object type %v is not supported", x)
	}
	return podSpec, nil
}

func GetEphemeralContainers(resource runtime.Object) ([]corev1.Container, error) {
	podSpec, err := GetPodSpec(resource)
	if err != nil {
		return nil, err
	}
	containers := []corev1.Container{}
	if podSpec != nil {
		for i := range podSpec.EphemeralContainers {
			// with this conversion, we loose some fields:
			// https://pkg.go.dev/k8s.io/api/core/v1#EphemeralContainer
			// https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#ephemeralcontainer-v1-core
			// but we don't need them for now
			// from what I can tell, we only loose: targetContainerName
			v1Container := corev1.Container(podSpec.EphemeralContainers[i].EphemeralContainerCommon)
			containers = append(containers, v1Container)
		}
	}
	return containers, nil
}

func GetInitContainers(resource runtime.Object) ([]corev1.Container, error) {
	podSpec, err := GetPodSpec(resource)
	if err != nil {
		return nil, err
	}
	containers := []corev1.Container{}
	if podSpec != nil {
		containers = append(containers, podSpec.InitContainers...)
	}
	return containers, nil
}

func GetContainers(resource runtime.Object) ([]corev1.Container, error) {
	podSpec, err := GetPodSpec(resource)
	if err != nil {
		return nil, err
	}
	containers := []corev1.Container{}
	if podSpec != nil {
		containers = append(containers, podSpec.Containers...)
	}
	return containers, nil
}

func FindByUid(resources []runtime.Object, uid string) (runtime.Object, error) {
	for i := range resources {
		res := resources[i]
		obj, err := meta.Accessor(res)
		if err != nil {
			log.Error().Err(err).Msg("could not access object attributes")
			continue
		}
		if string(obj.GetUID()) == uid {
			return res, nil
		}
	}
	return nil, errors.New("not found")
}
