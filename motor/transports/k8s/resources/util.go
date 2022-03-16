package resources

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

func ToCoreV1Pod(resource runtime.Object) (*corev1.Pod, error) {
	pod, ok := resource.(*corev1.Pod)
	if !ok {
		return nil, errors.New("could not convert to corev1.Pod")
	}
	return pod, nil
}

func ToAppsV1Deployment(resource runtime.Object) (*appsv1.Deployment, error) {
	deployment, ok := resource.(*appsv1.Deployment)
	if !ok {
		return nil, errors.New("could not convert to corev1.Pod")
	}
	return deployment, nil
}

func GetPodSpec(resource runtime.Object) *corev1.PodSpec {
	var podSpec *corev1.PodSpec

	switch x := resource.(type) {
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
	}
	return podSpec
}

func GetContainers(resource runtime.Object) []corev1.Container {
	podSpec := GetPodSpec(resource)
	containers := []corev1.Container{}
	if podSpec != nil {
		containers = append(containers, podSpec.InitContainers...)
		containers = append(containers, podSpec.Containers...)
	}
	return containers
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
